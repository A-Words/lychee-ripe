package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
)

// upstreamHealth is the shape of the FastAPI /v1/health response.
type upstreamHealth struct {
	Status string `json:"status"`
	Model  struct {
		ModelVersion  string `json:"model_version"`
		SchemaVersion string `json:"schema_version"`
		Adapter       string `json:"adapter"`
		Loaded        bool   `json:"loaded"`
	} `json:"model"`
}

// healthResponse is the gateway's aggregated health response.
type healthResponse struct {
	Status   string          `json:"status"`
	Gateway  string          `json:"gateway"`
	Upstream *upstreamHealth `json:"upstream,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Health returns an HTTP handler that checks its own liveness and the upstream
// FastAPI /v1/health endpoint, returning an aggregated status.
func Health(cfg config.UpstreamConfig, logger *slog.Logger) http.HandlerFunc {
	client := &http.Client{
		Timeout: time.Duration(cfg.TimeoutS) * time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		resp := healthResponse{
			Status:  "ok",
			Gateway: "ok",
		}

		upURL := fmt.Sprintf("%s/v1/health", cfg.BaseURL)
		upResp, err := client.Get(upURL)
		if err != nil {
			logger.Warn("healthz: upstream unreachable", "error", err)
			resp.Status = "degraded"
			resp.Error = "upstream unreachable"
		} else {
			defer upResp.Body.Close()
			body, _ := io.ReadAll(upResp.Body)
			var uh upstreamHealth
			if err := json.Unmarshal(body, &uh); err != nil {
				resp.Status = "degraded"
				resp.Error = "upstream response invalid"
			} else {
				resp.Upstream = &uh
				if uh.Status != "ok" {
					resp.Status = "degraded"
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if resp.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
