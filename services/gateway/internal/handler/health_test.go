package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lychee-ripe/gateway/internal/config"
	"log/slog"
)

func TestHealthUpstreamOK(t *testing.T) {
	// Fake upstream /v1/health.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","model":{"model_version":"1.0.0","schema_version":"v1","adapter":"yolo","loaded":true}}`))
	}))
	defer upstream.Close()

	cfg := config.UpstreamConfig{
		BaseURL:  upstream.URL,
		TimeoutS: 5,
	}
	logger := slog.Default()
	h := Health(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if resp.Upstream == nil {
		t.Fatal("upstream should not be nil")
	}
	if resp.Upstream.Model.ModelVersion != "1.0.0" {
		t.Errorf("model_version = %q", resp.Upstream.Model.ModelVersion)
	}
}

func TestHealthUpstreamDown(t *testing.T) {
	cfg := config.UpstreamConfig{
		BaseURL:  "http://127.0.0.1:1", // Nothing listening.
		TimeoutS: 1,
	}
	logger := slog.Default()
	h := Health(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}

	var resp healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "degraded" {
		t.Errorf("status = %q, want degraded", resp.Status)
	}
}
