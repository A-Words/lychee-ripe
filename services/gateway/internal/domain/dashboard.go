package domain

import "time"

type StatusDistribution struct {
	Anchored      int64
	PendingAnchor int64
	AnchorFailed  int64
}

type RipenessDistribution struct {
	Green int64
	Half  int64
	Red   int64
	Young int64
}

type RecentAnchorRecord struct {
	BatchID    string
	TraceCode  string
	Status     BatchStatus
	TxHash     *string
	AnchoredAt *time.Time
	CreatedAt  time.Time
}
