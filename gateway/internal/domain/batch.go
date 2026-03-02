package domain

import "time"

type BatchStatus string

const (
	BatchStatusPendingAnchor BatchStatus = "pending_anchor"
	BatchStatusAnchored      BatchStatus = "anchored"
	BatchStatusAnchorFailed  BatchStatus = "anchor_failed"
)

type UnripeHandling string

const (
	UnripeHandlingSortedOut UnripeHandling = "sorted_out"
)

type BatchSummary struct {
	Total          int
	Green          int
	Half           int
	Red            int
	Young          int
	UnripeCount    int
	UnripeRatio    float64
	UnripeHandling UnripeHandling
}

type AnchorProofRecord struct {
	TxHash          string
	BlockNumber     int64
	ChainID         string
	ContractAddress string
	AnchorHash      string
	AnchoredAt      time.Time
}

type BatchRecord struct {
	BatchID       string
	TraceCode     string
	Status        BatchStatus
	OrchardID     string
	OrchardName   string
	PlotID        string
	PlotName      *string
	HarvestedAt   time.Time
	Summary       BatchSummary
	Note          *string
	AnchorHash    *string
	ConfirmUnripe bool
	RetryCount    int
	LastError     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	AnchorProof   *AnchorProofRecord
}

type CreateBatchParams struct {
	BatchID       string
	TraceCode     string
	Status        BatchStatus
	OrchardID     string
	OrchardName   string
	PlotID        string
	PlotName      *string
	HarvestedAt   time.Time
	Summary       BatchSummary
	Note          *string
	AnchorHash    *string
	ConfirmUnripe bool
	RetryCount    int
	LastError     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
