package stripe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/pkg/logger"
)

// Client is a Stripe REST API client using HTTP Basic Auth.
type Client struct {
	secretKey     string
	webhookSecret string
	httpClient    *http.Client
}

// NewClient creates a new Stripe client.
func NewClient(cfg *config.StripeConfig) *Client {
	return &Client{
		secretKey:     cfg.SecretKey,
		webhookSecret: cfg.WebhookSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest sends an authenticated request to the Stripe API.
// POST requests use form-encoding; GET requests have no body.
func (c *Client) doRequest(method, path string, params url.Values) ([]byte, int, error) {
	var body io.Reader
	if method == http.MethodPost && params != nil {
		body = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequest(method, "https://api.stripe.com"+path, body)
	if err != nil {
		return nil, 0, fmt.Errorf("stripe: build request: %w", err)
	}
	req.SetBasicAuth(c.secretKey, "")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("stripe: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// CreatePaymentIntent creates a Stripe PaymentIntent and returns (clientSecret, piID, error).
// amountCents must be the charge amount in the smallest currency unit (e.g. cents for USD).
func (c *Client) CreatePaymentIntent(amountCents int64, currency, description string, metadata map[string]string) (string, string, error) {
	params := url.Values{}
	params.Set("amount", fmt.Sprintf("%d", amountCents))
	params.Set("currency", strings.ToLower(currency))
	params.Set("description", description)
	// automatic_payment_methods lets Stripe decide which methods to show (cards, wallets, etc.)
	params.Set("automatic_payment_methods[enabled]", "true")
	for k, v := range metadata {
		params.Set(fmt.Sprintf("metadata[%s]", k), v)
	}

	respBody, status, err := c.doRequest(http.MethodPost, "/v1/payment_intents", params)
	if err != nil {
		return "", "", err
	}

	if status != http.StatusOK {
		var errResp stripeErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return "", "", fmt.Errorf("stripe: create payment intent failed (status %d): %s", status, errResp.Error.Message)
	}

	var pi paymentIntentResponse
	if err := json.Unmarshal(respBody, &pi); err != nil {
		return "", "", fmt.Errorf("stripe: parse payment intent: %w", err)
	}

	logger.Info("Stripe PaymentIntent created", "pi_id", pi.ID, "amount", amountCents, "currency", currency)
	return pi.ClientSecret, pi.ID, nil
}

// RetrievePaymentIntent fetches a PaymentIntent by ID and returns its status.
// Use this server-side to verify the payment before marking an order as paid.
func (c *Client) RetrievePaymentIntent(piID string) (string, error) {
	respBody, status, err := c.doRequest(http.MethodGet, "/v1/payment_intents/"+piID, nil)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		var errResp stripeErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return "", fmt.Errorf("stripe: retrieve payment intent failed (status %d): %s", status, errResp.Error.Message)
	}

	var pi paymentIntentResponse
	if err := json.Unmarshal(respBody, &pi); err != nil {
		return "", fmt.Errorf("stripe: parse payment intent: %w", err)
	}

	return pi.Status, nil
}

// ---- internal types ----

type paymentIntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	// "requires_payment_method" | "requires_confirmation" | "requires_action"
	// "processing" | "requires_capture" | "canceled" | "succeeded"
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type stripeErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
