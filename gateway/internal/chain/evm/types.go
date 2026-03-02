package evm

import "time"

type AnchorBatchRequest struct {
	BatchID    string
	AnchorHash string
	Timestamp  time.Time
}

type BatchAnchorOnChain struct {
	BatchID    string
	AnchorHash string
	AnchoredAt time.Time
}
