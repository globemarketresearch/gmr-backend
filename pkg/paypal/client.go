package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/pkg/logger"
)

// Client is a PayPal REST API client with token caching.
type Client struct {
	cfg          *config.PayPalConfig
	baseURL      string
	httpClient   *http.Client
	mu           sync.Mutex
	accessToken  string
	tokenExpires time.Time
}

// NewClient creates a new PayPal client.
func NewClient(cfg *config.PayPalConfig) *Client {
	baseURL := "https://api-m.sandbox.paypal.com"
	if cfg.Mode == "live" {
		baseURL = "https://api-m.paypal.com"
	}

	return &Client{
		cfg:     cfg,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getToken returns a valid access token, fetching a new one if needed.
func (c *Client) getToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Refresh 5 minutes before expiry
	if c.accessToken != "" && time.Now().Before(c.tokenExpires.Add(-5*time.Minute)) {
		return c.accessToken, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("paypal: build token request: %w", err)
	}
	req.SetBasicAuth(c.cfg.ClientID, c.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("paypal: fetch token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("paypal: token request failed (status %d): %s", resp.StatusCode, body)
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("paypal: parse token response: %w", err)
	}

	c.accessToken = tok.AccessToken
	// PayPal tokens typically last 9 hours (32400 seconds)
	c.tokenExpires = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	logger.Info("PayPal access token refreshed", "expires_in_seconds", tok.ExpiresIn)
	return c.accessToken, nil
}

// doRequest executes an authenticated JSON request against the PayPal API.
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, 0, err
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("paypal: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("paypal: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("paypal: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// CreateOrder creates a PayPal order and returns the PayPal order ID.
func (c *Client) CreateOrder(amount string, currency, description, referenceID string) (*CreateOrderResponse, error) {
	reqBody := CreateOrderRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []PurchaseUnit{
			{
				ReferenceID: referenceID,
				Description: description,
				Amount: Amount{
					CurrencyCode: currency,
					Value:        amount,
				},
			},
		},
	}

	respBody, status, err := c.doRequest(http.MethodPost, "/v2/checkout/orders", reqBody)
	if err != nil {
		return nil, err
	}

	if status != http.StatusCreated {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("paypal: create order failed (status %d): %s - %s", status, errResp.Name, errResp.Message)
	}

	var result CreateOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("paypal: parse create order response: %w", err)
	}

	return &result, nil
}

// CaptureOrder captures a PayPal order and returns the capture ID.
func (c *Client) CaptureOrder(paypalOrderID string) (captureID string, status string, err error) {
	respBody, httpStatus, doErr := c.doRequest(
		http.MethodPost,
		fmt.Sprintf("/v2/checkout/orders/%s/capture", paypalOrderID),
		struct{}{},
	)
	if doErr != nil {
		return "", "", doErr
	}

	if httpStatus != http.StatusCreated && httpStatus != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return "", "", fmt.Errorf("paypal: capture order failed (status %d): %s - %s", httpStatus, errResp.Name, errResp.Message)
	}

	// Parse capture response — captures live inside purchase_units[0].payments.captures[0]
	var raw struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []Capture `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return "", "", fmt.Errorf("paypal: parse capture response: %w", err)
	}

	if len(raw.PurchaseUnits) > 0 && len(raw.PurchaseUnits[0].Payments.Captures) > 0 {
		capture := raw.PurchaseUnits[0].Payments.Captures[0]
		return capture.ID, capture.Status, nil
	}

	// Fallback: order-level status
	return "", raw.Status, nil
}
