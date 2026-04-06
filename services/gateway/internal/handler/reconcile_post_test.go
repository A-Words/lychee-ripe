package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/service"
)

func TestReconcileBatchesReturns202(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{
		result: service.ReconcileResult{
			Accepted:       true,
			RequestedCount: 3,
			ScheduledCount: 1,
			SkippedCount:   2,
			Message:        "reconcile accepted",
		},
	}

	body := `{"batch_ids":["batch_1","missing","batch_2"],"limit":100}`
	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	ReconcileBatches(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["accepted"] != true {
		t.Fatalf("accepted = %v, want true", resp["accepted"])
	}
	if resp["requested_count"] != float64(3) || resp["scheduled_count"] != float64(1) || resp["skipped_count"] != float64(2) {
		t.Fatalf("unexpected counts: %+v", resp)
	}
	if resp["message"] != "reconcile accepted" {
		t.Fatalf("message = %v, want reconcile accepted", resp["message"])
	}
}

func TestReconcileBatchesAllowsEmptyBody(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{
		result: service.ReconcileResult{Accepted: true, Message: "reconcile accepted"},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", nil)
	rec := httptest.NewRecorder()
	ReconcileBatches(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if svc.input.Limit != nil {
		t.Fatalf("limit = %v, want nil", *svc.input.Limit)
	}
	if len(svc.input.BatchIDs) != 0 {
		t.Fatalf("batch_ids = %v, want empty", svc.input.BatchIDs)
	}
}

func TestReconcileBatchesReturns400ForInvalidLimit(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{
		fn: func(_ context.Context, input service.ManualReconcileInput) (service.ReconcileResult, error) {
			if input.Limit != nil && *input.Limit == 0 {
				return service.ReconcileResult{}, service.ErrInvalidRequest
			}
			return service.ReconcileResult{}, nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", bytes.NewBufferString(`{"limit":0}`))
	rec := httptest.NewRecorder()
	ReconcileBatches(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestReconcileBatchesReturns404(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{err: service.ErrNotFound}
	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	ReconcileBatches(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] != "pending_batch_not_found" {
		t.Fatalf("error = %v, want pending_batch_not_found", resp["error"])
	}
}

func TestReconcileBatchesReturns409(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{err: service.ErrConflict}
	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	ReconcileBatches(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] != "trace_mode_conflict" {
		t.Fatalf("error = %v, want trace_mode_conflict", resp["error"])
	}
}

func TestReconcileBatchesReturns503(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{err: service.ErrServiceUnavailable}
	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	ReconcileBatches(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestReconcileBatchesErrorContainsRequestID(t *testing.T) {
	t.Parallel()

	svc := &fakeReconcileService{err: service.ErrNotFound}
	req := httptest.NewRequest(http.MethodPost, "/v1/batches/reconcile", bytes.NewBufferString(`{}`))
	req.Header.Set("X-Request-ID", "rid-reconcile-404")
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(ReconcileBatches(svc, nil))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["request_id"] != "rid-reconcile-404" {
		t.Fatalf("request_id = %v, want rid-reconcile-404", resp["request_id"])
	}
}

type fakeReconcileService struct {
	result service.ReconcileResult
	err    error
	input  service.ManualReconcileInput
	fn     func(ctx context.Context, input service.ManualReconcileInput) (service.ReconcileResult, error)
}

func (f *fakeReconcileService) TriggerManualReconcile(ctx context.Context, input service.ManualReconcileInput) (service.ReconcileResult, error) {
	f.input = input
	if f.fn != nil {
		return f.fn(ctx, input)
	}
	if f.err != nil {
		return service.ReconcileResult{}, f.err
	}
	return f.result, nil
}
