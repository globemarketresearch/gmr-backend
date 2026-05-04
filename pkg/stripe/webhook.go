package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// maxWebhookAge is the replay-attack window Stripe recommends (5 minutes).
const maxWebhookAge = 5 * time.Minute

// VerifyWebhookSignature validates the Stripe-Signature header against the raw
// request body using the webhook secret. Returns an error if the signature is
// invalid or the event is older than maxWebhookAge.
//
// Stripe signature format: t=<unix_ts>,v1=<hex_hmac>[,v1=<hex_hmac>,...]
func (c *Client) VerifyWebhookSignature(payload []byte, sigHeader string) error {
	if c.webhookSecret == "" {
		return fmt.Errorf("stripe: webhook secret not configured")
	}

	var timestamp int64
	var v1Sigs []string

	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return fmt.Errorf("stripe: invalid timestamp in signature header")
			}
			timestamp = ts
		case "v1":
			v1Sigs = append(v1Sigs, kv[1])
		}
	}

	if timestamp == 0 {
		return fmt.Errorf("stripe: no timestamp in signature header")
	}
	if len(v1Sigs) == 0 {
		return fmt.Errorf("stripe: no v1 signature in header")
	}

	// Replay-attack guard
	age := time.Since(time.Unix(timestamp, 0))
	if age > maxWebhookAge || age < -maxWebhookAge {
		return fmt.Errorf("stripe: webhook timestamp too old (age: %v)", age)
	}

	// Signed payload = "<timestamp>.<raw_body>"
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range v1Sigs {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}

	return fmt.Errorf("stripe: webhook signature mismatch")
}

// ParseWebhookEvent parses the raw webhook payload into a WebhookEvent.
func ParseWebhookEvent(payload []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("stripe: parse webhook event: %w", err)
	}
	return &event, nil
}

// WebhookEvent is a minimal representation of a Stripe event object.
type WebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// PaymentIntentFromEvent extracts a PaymentIntent from a webhook event's data.
func PaymentIntentFromEvent(event *WebhookEvent) (*paymentIntentResponse, error) {
	var pi paymentIntentResponse
	if err := json.Unmarshal(event.Data.Object, &pi); err != nil {
		return nil, fmt.Errorf("stripe: parse payment intent from event: %w", err)
	}
	return &pi, nil
}
