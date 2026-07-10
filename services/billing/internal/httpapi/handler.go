// Package httpapi exposes billing over HTTP: per-customer invoices with margin,
// and usage export to the billing provider.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/marginpilot/billing/internal/cogs"
	"github.com/marginpilot/billing/internal/store"
	"github.com/marginpilot/billing/internal/stripe"
)

// Handler serves the billing API.
type Handler struct {
	subs *store.Store
	cogs *cogs.Reader
	exp  stripe.Exporter
	log  *slog.Logger
}

// New constructs the handler.
func New(subs *store.Store, c *cogs.Reader, exp stripe.Exporter, log *slog.Logger) *Handler {
	return &Handler{subs: subs, cogs: c, exp: exp, log: log}
}

// Register wires routes onto the mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/invoice/{tenant}", h.invoiceTenant)
	mux.HandleFunc("GET /v1/invoice/{tenant}/{customer}", h.invoiceCustomer)
	mux.HandleFunc("POST /v1/export/{tenant}/{customer}", h.export)
}

type line struct {
	CustomerID     string   `json:"customer_id"`
	Plan           string   `json:"plan"`
	RevenueMicros  int64    `json:"revenue_micros"`
	CogsMicros     int64    `json:"cogs_micros"`
	GrossMarginPct *float64 `json:"gross_margin_pct"`
}

func (h *Handler) invoiceTenant(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")

	subs, err := h.subs.List(r.Context(), tenant)
	if err != nil {
		h.fail(w, "list subscriptions", err)
		return
	}
	costs, err := h.cogs.ByCustomer(r.Context(), tenant)
	if err != nil {
		h.fail(w, "read cogs", err)
		return
	}

	lines := make([]line, 0, len(subs))
	for _, s := range subs {
		c := costs[s.CustomerID]
		lines = append(lines, line{
			CustomerID:     s.CustomerID,
			Plan:           s.Plan,
			RevenueMicros:  s.RevenueMicros,
			CogsMicros:     c,
			GrossMarginPct: marginPct(s.RevenueMicros, c),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": tenant, "lines": lines})
}

func (h *Handler) invoiceCustomer(w http.ResponseWriter, r *http.Request) {
	tenant, customer := r.PathValue("tenant"), r.PathValue("customer")

	sub, found, err := h.subs.Get(r.Context(), tenant, customer)
	if err != nil {
		h.fail(w, "get subscription", err)
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "customer not found"})
		return
	}
	c, err := h.cogs.Customer(r.Context(), tenant, customer)
	if err != nil {
		h.fail(w, "read cogs", err)
		return
	}
	writeJSON(w, http.StatusOK, line{
		CustomerID:     sub.CustomerID,
		Plan:           sub.Plan,
		RevenueMicros:  sub.RevenueMicros,
		CogsMicros:     c,
		GrossMarginPct: marginPct(sub.RevenueMicros, c),
	})
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	tenant, customer := r.PathValue("tenant"), r.PathValue("customer")

	c, err := h.cogs.Customer(r.Context(), tenant, customer)
	if err != nil {
		h.fail(w, "read cogs", err)
		return
	}
	if err := h.exp.ExportUsage(r.Context(), tenant, customer, c); err != nil {
		h.fail(w, "export usage", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"exported": true, "tenant_id": tenant, "customer_id": customer, "usage_micros": c,
	})
}

func marginPct(revenue, cogs int64) *float64 {
	if revenue <= 0 {
		return nil
	}
	v := float64(revenue-cogs) / float64(revenue) * 100
	return &v
}

func (h *Handler) fail(w http.ResponseWriter, what string, err error) {
	h.log.Error(what+" failed", "err", err)
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": what + " failed"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
