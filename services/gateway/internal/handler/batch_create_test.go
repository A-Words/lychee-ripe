package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/service"
)

func TestCreateBatchReturns201(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchCreateService{
		result: service.CreateBatchResult{
			Batch:      sampleDomainBatch(domain.BatchStatusAnchored, true),
			HTTPStatus: http.StatusCreated,
		},
	}

	body := `{
		"orchard_id":"orchard-1",
		"orchard_name":"orchard",
		"plot_id":"plot-1",
		"harvested_at":"2026-03-02T10:30:00Z",
		"summary":{"total":10,"green":1,"half":3,"red":6,"young":0}
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/batches", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	CreateBatch(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["batch_id"] != "batch_01" {
		t.Fatalf("batch_id = %v, want batch_01", resp["batch_id"])
	}
	if resp["trace_code"] != "TRC-ABCD-EFGH" {
		t.Fatalf("trace_code = %v, want TRC-ABCD-EFGH", resp["trace_code"])
	}
	if resp["anchor_proof"] == nil {
		t.Fatal("anchor_proof should not be nil")
	}
}

func TestCreateBatchReturns202(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchCreateService{
		result: service.CreateBatchResult{
			Batch:      sampleDomainBatch(domain.BatchStatusPendingAnchor, false),
			HTTPStatus: http.StatusAccepted,
		},
	}

	body := `{
		"orchard_id":"orchard-1",
		"orchard_name":"orchard",
		"plot_id":"plot-1",
		"harvested_at":"2026-03-02T10:30:00Z",
		"summary":{"total":10,"green":1,"half":3,"red":6,"young":0}
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/batches", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	CreateBatch(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["anchor_proof"] != nil {
		t.Fatalf("anchor_proof = %v, want nil", resp["anchor_proof"])
	}
}

func TestCreateBatchErrorStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid request", err: service.ErrInvalidRequest, wantStatus: http.StatusBadRequest},
		{name: "conflict", err: service.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unavailable", err: service.ErrServiceUnavailable, wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeBatchCreateService{err: tt.err}
			body := `{
				"orchard_id":"orchard-1",
				"orchard_name":"orchard",
				"plot_id":"plot-1",
				"harvested_at":"2026-03-02T10:30:00Z",
				"summary":{"total":10,"green":1,"half":3,"red":6,"young":0}
			}`
			req := httptest.NewRequest(http.MethodPost, "/v1/batches", bytes.NewBufferString(body))
			rec := httptest.NewRecorder()

			CreateBatch(svc, nil).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestCreateBatchErrorContainsRequestID(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchCreateService{err: service.ErrInvalidRequest}
	body := `{
		"orchard_id":"orchard-1",
		"orchard_name":"orchard",
		"plot_id":"plot-1",
		"harvested_at":"2026-03-02T10:30:00Z",
		"summary":{"total":10,"green":1,"half":3,"red":6,"young":0}
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/batches", bytes.NewBufferString(body))
	req.Header.Set("X-Request-ID", "rid-123")
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(CreateBatch(svc, nil))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["request_id"] != "rid-123" {
		t.Fatalf("request_id = %v, want rid-123", resp["request_id"])
	}
}

func TestCreateBatchRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	svc := &fakeBatchCreateService{}
	req := httptest.NewRequest(http.MethodPost, "/v1/batches", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	CreateBatch(svc, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

type fakeBatchCreateService struct {
	result service.CreateBatchResult
	err    error
}

func (f *fakeBatchCreateService) CreateBatch(_ context.Context, _ service.BatchCreateInput) (service.CreateBatchResult, error) {
	if f.err != nil {
		return service.CreateBatchResult{}, f.err
	}
	return f.result, nil
}

func sampleDomainBatch(status domain.BatchStatus, withProof bool) domain.BatchRecord {
	createdAt := time.Date(2026, 3, 2, 10, 31, 2, 0, time.UTC)
	record := domain.BatchRecord{
		BatchID:     "batch_01",
		TraceCode:   "TRC-ABCD-EFGH",
		TraceMode:   domain.TraceModeBlockchain,
		Status:      status,
		OrchardID:   "orchard-1",
		OrchardName: "orchard",
		PlotID:      "plot-1",
		HarvestedAt: time.Date(2026, 3, 2, 10, 30, 0, 0, time.UTC),
		Summary: domain.BatchSummary{
			Total:          10,
			Green:          1,
			Half:           3,
			Red:            6,
			Young:          0,
			UnripeCount:    1,
			UnripeRatio:    0.1,
			UnripeHandling: domain.UnripeHandlingSortedOut,
		},
		CreatedAt: createdAt,
	}
	if withProof {
		record.AnchorProof = &domain.AnchorProofRecord{
			TxHash:          "0xabc",
			BlockNumber:     100,
			ChainID:         "31337",
			ContractAddress: "0xdef",
			AnchorHash:      "0xhash",
			AnchoredAt:      createdAt,
		}
	}
	return record
}
