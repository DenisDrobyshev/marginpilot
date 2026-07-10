// Command dashboard is a tiny BFF (backend-for-frontend): it serves a single
// HTML console and, server-side, aggregates the billing and analytics APIs so
// the browser makes only same-origin calls (no CORS). A "simulate" endpoint
// sends a request through the gateway to generate live usage.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"
)

//go:embed web/index.html
var indexHTML []byte

type server struct {
	gateway   string
	billing   string
	analytics string
	client    *http.Client
	log       *logger.Logger
}

func main() {
	log := logger.New("dashboard")
	s := &server{
		gateway:   config.Get("GATEWAY_URL", "http://localhost:18080"),
		billing:   config.Get("BILLING_URL", "http://localhost:18085"),
		analytics: config.Get("ANALYTICS_URL", "http://localhost:18000"),
		client:    &http.Client{Timeout: 8 * time.Second},
		log:       log,
	}

	srv := httpx.New(config.Get("DASHBOARD_ADDR", ":8087"), log)
	mux := srv.Mux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/overview", s.overview)
	mux.HandleFunc("POST /api/simulate", s.simulate)

	if err := srv.Run(context.Background()); err != nil {
		log.Error("server exited with error", "err", err)
	}
}

// overview aggregates billing + analytics for a tenant into one payload.
func (s *server) overview(w http.ResponseWriter, r *http.Request) {
	tenant := tenantOf(r)
	writeJSON(w, http.StatusOK, map[string]json.RawMessage{
		"invoice":   s.get(r.Context(), s.billing+"/v1/invoice/"+tenant),
		"forecast":  s.get(r.Context(), s.analytics+"/v1/forecast/"+tenant),
		"anomalies": s.get(r.Context(), s.analytics+"/v1/anomalies/"+tenant),
		"usage":     s.get(r.Context(), s.analytics+"/v1/usage_timeseries/"+tenant),
	})
}

// simulate sends one chat request through the gateway to generate usage.
func (s *server) simulate(w http.ResponseWriter, r *http.Request) {
	tenant := tenantOf(r)
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"dashboard demo ` +
		time.Now().Format("15:04:05.000") + `"}]}`
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost,
		s.gateway+"/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+keyFor(tenant))
	req.Header.Set("Content-Type", "application/json")

	code := 0
	if resp, err := s.client.Do(req); err == nil {
		code = resp.StatusCode
		_ = resp.Body.Close()
	} else {
		s.log.Error("simulate failed", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenant": tenant, "gateway_status": code})
}

// get fetches a JSON URL, returning the body or a small {"error":...} document.
func (s *server) get(ctx context.Context, url string) json.RawMessage {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return b
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 || len(b) == 0 {
		e, _ := json.Marshal(map[string]string{"error": resp.Status})
		return e
	}
	return b
}

func tenantOf(r *http.Request) string {
	if t := r.URL.Query().Get("tenant"); t != "" {
		return t
	}
	return "demo-tenant"
}

// keyFor maps a tenant to its seeded virtual API key for demo traffic.
func keyFor(tenant string) string {
	if tenant == "northwind" {
		return "sk-north"
	}
	return "sk-demo"
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
