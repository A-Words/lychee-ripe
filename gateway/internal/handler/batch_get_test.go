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

func TestGetBatchReturns200WithExpectedFields(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchGetService{
		record: sampleDomainBatch(domain.BatchStatusAnchored, true),
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/batches/batch_01", nil)
	req.SetPathValue("batch_id", "batch_01")
	rec := httptest.NewRecorder()

	GetBatch(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["batch_id"] != "batch_01" {
		t.Fatalf("batch_id = %v, want batch_01", resp["batch_id"])
	}
	summary, ok := resp["summary"].(map[string]any)
	if !ok {
		t.Fatal("summary should be an object")
	}
	if summary["unripe_count"] == nil || summary["unripe_ratio"] == nil || summary["unripe_handling"] == nil {
		t.Fatalf("summary missing unripe fields: %+v", summary)
	}
	if resp["anchor_proof"] == nil {
		t.Fatal("anchor_proof should not be nil")
	}
}

func TestGetBatchReturns404(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchGetService{err: service.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/v1/batches/not_found", nil)
	req.SetPathValue("batch_id", "not_found")
	rec := httptest.NewRecorder()

	GetBatch(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetBatchReturns503(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchGetService{err: service.ErrServiceUnavailable}
	req := httptest.NewRequest(http.MethodGet, "/v1/batches/error", nil)
	req.SetPathValue("batch_id", "error")
	rec := httptest.NewRecorder()

	GetBatch(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestGetBatchErrorContainsRequestID(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchGetService{err: service.ErrNotFound}
	req := httptest.NewRequest(http.MethodGet, "/v1/batches/not_found", nil)
	req.SetPathValue("batch_id", "not_found")
	req.Header.Set("X-Request-ID", "rid-404")
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(GetBatch(svc, nil))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["request_id"] != "rid-404" {
		t.Fatalf("request_id = %v, want rid-404", resp["request_id"])
	}
}

type fakeBatchGetService struct {
	record domain.BatchRecord
	err    error
}

func (f *fakeBatchGetService) GetBatchByID(_ context.Context, _ string) (domain.BatchRecord, error) {
	if f.err != nil {
		return domain.BatchRecord{}, f.err
	}
	return f.record, nil
}
