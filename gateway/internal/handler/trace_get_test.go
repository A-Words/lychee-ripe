package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/service"
)

func TestGetPublicTraceReturns200WithExpectedSchema(t *testing.T) {
	t.Parallel()

	record := sampleDomainBatch(domain.BatchStatusAnchored, true)
	record.PlotName = nil
	svc := &fakeTraceGetService{
		result: service.TraceQueryResult{
			Batch:        record,
			VerifyStatus: service.TraceVerifyStatusPass,
			Reason:       "anchor_hash matches on-chain record",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/trace/TRC-ABCD-EFGH", nil)
	req.SetPathValue("trace_code", "TRC-ABCD-EFGH")
	rec := httptest.NewRecorder()

	GetPublicTrace(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	batch, ok := resp["batch"].(map[string]any)
	if !ok {
		t.Fatal("batch should be an object")
	}
	if batch["batch_id"] != "batch_01" {
		t.Fatalf("batch_id = %v, want batch_01", batch["batch_id"])
	}
	if batch["plot_name"] != "" {
		t.Fatalf("plot_name = %v, want empty string", batch["plot_name"])
	}
	summary, ok := batch["summary"].(map[string]any)
	if !ok {
		t.Fatal("summary should be an object")
	}
	if summary["unripe_count"] == nil || summary["unripe_ratio"] == nil || summary["unripe_handling"] == nil {
		t.Fatalf("summary missing unripe fields: %+v", summary)
	}
	verify, ok := resp["verify_result"].(map[string]any)
	if !ok {
		t.Fatal("verify_result should be an object")
	}
	if verify["verify_status"] != "pass" {
		t.Fatalf("verify_status = %v, want pass", verify["verify_status"])
	}
	if verify["reason"] != "anchor_hash matches on-chain record" {
		t.Fatalf("reason = %v, want match message", verify["reason"])
	}
}

func TestGetPublicTraceVerifyStatesAndReasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		verifyStatus string
		reason       string
	}{
		{
			name:         "pass",
			verifyStatus: service.TraceVerifyStatusPass,
			reason:       "anchor_hash matches on-chain record",
		},
		{
			name:         "fail",
			verifyStatus: service.TraceVerifyStatusFail,
			reason:       "anchor_hash does not match on-chain record",
		},
		{
			name:         "pending",
			verifyStatus: service.TraceVerifyStatusPending,
			reason:       "batch is not anchored yet",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeTraceGetService{
				result: service.TraceQueryResult{
					Batch:        sampleDomainBatch(domain.BatchStatusAnchored, true),
					VerifyStatus: tt.verifyStatus,
					Reason:       tt.reason,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/v1/trace/TRC-ABCD-EFGH", nil)
			req.SetPathValue("trace_code", "TRC-ABCD-EFGH")
			rec := httptest.NewRecorder()

			GetPublicTrace(svc, nil).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var resp map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			verify := resp["verify_result"].(map[string]any)
			if verify["verify_status"] != tt.verifyStatus {
				t.Fatalf("verify_status = %v, want %s", verify["verify_status"], tt.verifyStatus)
			}
			if verify["reason"] != tt.reason {
				t.Fatalf("reason = %v, want %s", verify["reason"], tt.reason)
			}
		})
	}
}

func TestGetPublicTraceReturns404(t *testing.T) {
	t.Parallel()

	svc := &fakeTraceGetService{err: service.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/v1/trace/unknown", nil)
	req.SetPathValue("trace_code", "unknown")
	rec := httptest.NewRecorder()

	GetPublicTrace(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] != "trace_not_found" {
		t.Fatalf("error = %v, want trace_not_found", resp["error"])
	}
}

func TestGetPublicTraceReturns503(t *testing.T) {
	t.Parallel()

	svc := &fakeTraceGetService{err: service.ErrServiceUnavailable}
	req := httptest.NewRequest(http.MethodGet, "/v1/trace/error", nil)
	req.SetPathValue("trace_code", "error")
	rec := httptest.NewRecorder()

	GetPublicTrace(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestGetPublicTraceErrorContainsRequestID(t *testing.T) {
	t.Parallel()

	svc := &fakeTraceGetService{err: service.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/v1/trace/unknown", nil)
	req.SetPathValue("trace_code", "unknown")
	req.Header.Set("X-Request-ID", "rid-trace-404")
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(GetPublicTrace(svc, nil))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["request_id"] != "rid-trace-404" {
		t.Fatalf("request_id = %v, want rid-trace-404", resp["request_id"])
	}
}

type fakeTraceGetService struct {
	result service.TraceQueryResult
	err    error
}

func (f *fakeTraceGetService) GetPublicTrace(_ context.Context, _ string) (service.TraceQueryResult, error) {
	if f.err != nil {
		return service.TraceQueryResult{}, f.err
	}
	return f.result, nil
}
