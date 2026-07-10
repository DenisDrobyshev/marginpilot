package stripe

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTP pushes usage to Stripe's meter events API. It is selected when
// STRIPE_API_KEY is set. Production use also needs a customer_id -> Stripe
// customer mapping; here the customer_id is sent as-is in the payload.
type HTTP struct {
	apiKey    string
	meterName string
	client    *http.Client
	log       *slog.Logger
}

// NewHTTP builds the live Stripe exporter.
func NewHTTP(apiKey, meterName string, log *slog.Logger) *HTTP {
	return &HTTP{
		apiKey:    apiKey,
		meterName: meterName,
		client:    &http.Client{Timeout: 10 * time.Second},
		log:       log,
	}
}

// ExportUsage records a meter event for the customer's accrued usage.
func (h *HTTP) ExportUsage(ctx context.Context, tenant, customer string, costMicros int64) error {
	form := url.Values{}
	form.Set("event_name", h.meterName)
	form.Set("payload[stripe_customer_id]", customer)
	form.Set("payload[value]", fmt.Sprintf("%d", costMicros))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.stripe.com/v1/billing/meter_events", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(h.apiKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("stripe returned %s", resp.Status)
	}
	h.log.Info("stripe export", "tenant", tenant, "customer", customer, "usage_micros", costMicros)
	return nil
}
