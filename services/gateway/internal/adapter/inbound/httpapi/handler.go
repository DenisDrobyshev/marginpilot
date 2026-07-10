// Package httpapi is the inbound adapter that exposes the gateway core over an
// OpenAI-compatible HTTP API. It owns transport concerns only: auth extraction,
// decoding, status codes and encoding.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/marginpilot/gateway/internal/app"
	"github.com/marginpilot/gateway/internal/domain"
)

var errUnauthorized = errors.New("unauthorized")

// Handler adapts HTTP to the application core.
type Handler struct {
	svc      *app.Service
	resolver app.CallerResolver // may be nil -> header/demo fallback
	log      *slog.Logger
}

// New constructs the handler. A nil resolver enables header-based caller
// resolution for local/demo runs without an identity service.
func New(svc *app.Service, resolver app.CallerResolver, log *slog.Logger) *Handler {
	return &Handler{svc: svc, resolver: resolver, log: log}
}

// Register wires routes onto the mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/chat/completions", h.chatCompletions)
}

func (h *Handler) chatCompletions(w http.ResponseWriter, r *http.Request) {
	caller, err := h.resolveCaller(r.Context(), r)
	if err != nil {
		if errors.Is(err, errUnauthorized) || errors.Is(err, app.ErrKeyNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid or missing api key")
			return
		}
		h.log.Error("resolve caller failed", "err", err)
		writeError(w, http.StatusBadGateway, "identity service error")
		return
	}

	var req domain.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Proxy(r.Context(), caller, req)
	switch {
	case errors.Is(err, app.ErrBlocked):
		writeError(w, http.StatusForbidden, "request blocked by guardrail")
		return
	case errors.Is(err, app.ErrBudgetExceeded):
		writeError(w, http.StatusPaymentRequired, "budget exceeded for customer")
		return
	case err != nil:
		h.log.Error("proxy failed", "err", err)
		writeError(w, http.StatusBadGateway, "upstream provider error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// resolveCaller turns the request's API key into a caller. With an identity
// resolver it delegates the key lookup; without one it reads caller headers
// with demo defaults so the path is exercisable with a single curl.
func (h *Handler) resolveCaller(ctx context.Context, r *http.Request) (app.Caller, error) {
	key := bearer(r.Header.Get("Authorization"))
	if key == "" {
		return app.Caller{}, errUnauthorized
	}
	if h.resolver != nil {
		return h.resolver.Resolve(ctx, key)
	}
	return app.Caller{
		TenantID:   headerOr(r, "X-Tenant-Id", "demo-tenant"),
		CustomerID: headerOr(r, "X-Customer-Id", "demo-customer"),
		Feature:    headerOr(r, "X-Feature", "default"),
	}, nil
}

func bearer(h string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return strings.TrimSpace(h)
}

func headerOr(r *http.Request, key, def string) string {
	if v := r.Header.Get(key); v != "" {
		return v
	}
	return def
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": msg},
	})
}
