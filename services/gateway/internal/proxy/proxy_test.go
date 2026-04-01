package proxy

import (
	"bufio"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lychee-ripe/gateway/internal/config"
)

func TestWebSocketHostRewrite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != r.Header.Get("X-Expected-Host") {
			t.Errorf("upstream saw Host = %q, want %q", r.Host, r.Header.Get("X-Expected-Host"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := config.UpstreamConfig{
		BaseURL:  upstream.URL,
		TimeoutS: 5,
	}
	h, err := New(cfg, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/infer/stream", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	upstreamHost := strings.TrimPrefix(upstream.URL, "http://")
	req.Header.Set("X-Expected-Host", upstreamHost)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if req.Host != upstreamHost {
		t.Errorf("request Host was not rewritten: got %q, want %q", req.Host, upstreamHost)
	}
	if req.URL.Host != upstreamHost {
		t.Errorf("request URL.Host was not rewritten: got %q, want %q", req.URL.Host, upstreamHost)
	}
}

func TestIsWebSocket(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		upgrade    string
		want       bool
	}{
		{"valid", "Upgrade", "websocket", true},
		{"case-insensitive", "upgrade", "WebSocket", true},
		{"no-upgrade-header", "keep-alive", "", false},
		{"no-connection-header", "", "websocket", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.connection != "" {
				req.Header.Set("Connection", tt.connection)
			}
			if tt.upgrade != "" {
				req.Header.Set("Upgrade", tt.upgrade)
			}
			if got := isWebSocket(req); got != tt.want {
				t.Errorf("isWebSocket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPProxyForwarding(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	cfg := config.UpstreamConfig{
		BaseURL:  upstream.URL,
		TimeoutS: 5,
	}
	h, err := New(cfg, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ok") {
		t.Errorf("body = %q, want to contain 'ok'", body)
	}
}

// readHTTPRequest parses a raw HTTP request from a reader (test helper).
func readHTTPRequest(raw string) (*http.Request, error) {
	return http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
}
