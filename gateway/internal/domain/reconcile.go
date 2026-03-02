package domain

import "time"

type ReconcileTriggerType string

const (
	ReconcileTriggerManual ReconcileTriggerType = "manual"
	ReconcileTriggerAuto   ReconcileTriggerType = "auto"
)

type ReconcileJobStatus string

const (
	ReconcileJobStatusAccepted  ReconcileJobStatus = "accepted"
	ReconcileJobStatusRunning   ReconcileJobStatus = "running"
	ReconcileJobStatusCompleted ReconcileJobStatus = "completed"
	ReconcileJobStatusFailed    ReconcileJobStatus = "failed"
)

type CreateReconcileJobParams struct {
	JobID          string
	TriggerType    ReconcileTriggerType
	Status         ReconcileJobStatus
	RequestedCount int
	ScheduledCount int
	SkippedCount   int
	ErrorMessage   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ReconcileJobRecord struct {
	JobID          string
	TriggerType    ReconcileTriggerType
	Status         ReconcileJobStatus
	RequestedCount int
	ScheduledCount int
	SkippedCount   int
	ErrorMessage   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ReconcileJobItemRecord struct {
	BatchID      string
	BeforeStatus BatchStatus
	AfterStatus  BatchStatus
	AttemptNo    int
	ErrorMessage *string
	CreatedAt    time.Time
}

type ReconcileStats struct {
	PendingCount    int64
	RetriedTotal    int64
	FailedTotal     int64
	LastReconcileAt *time.Time
}
