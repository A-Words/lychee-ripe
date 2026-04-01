package gorm

import "time"

type BatchModel struct {
	ID             uint              `gorm:"primaryKey;autoIncrement"`
	BatchID        string            `gorm:"column:batch_id;type:text;not null;uniqueIndex"`
	TraceCode      string            `gorm:"column:trace_code;type:text;not null;uniqueIndex"`
	Status         string            `gorm:"column:status;type:text;not null;index:idx_batches_status_created_at,priority:1"`
	OrchardID      string            `gorm:"column:orchard_id;type:text;not null"`
	OrchardName    string            `gorm:"column:orchard_name;type:text;not null"`
	PlotID         string            `gorm:"column:plot_id;type:text;not null"`
	PlotName       *string           `gorm:"column:plot_name;type:text"`
	HarvestedAt    time.Time         `gorm:"column:harvested_at;not null"`
	Total          int               `gorm:"column:total;not null"`
	Green          int               `gorm:"column:green;not null"`
	Half           int               `gorm:"column:half;not null"`
	Red            int               `gorm:"column:red;not null"`
	Young          int               `gorm:"column:young;not null"`
	UnripeCount    int               `gorm:"column:unripe_count;not null"`
	UnripeRatio    float64           `gorm:"column:unripe_ratio;not null"`
	UnripeHandling string            `gorm:"column:unripe_handling;type:text;not null;default:sorted_out"`
	Note           *string           `gorm:"column:note;type:text"`
	AnchorHash     *string           `gorm:"column:anchor_hash;type:text"`
	ConfirmUnripe  bool              `gorm:"column:confirm_unripe;not null;default:false"`
	RetryCount     int               `gorm:"column:retry_count;not null;default:0"`
	LastError      *string           `gorm:"column:last_error;type:text"`
	CreatedAt      time.Time         `gorm:"column:created_at;not null;index:idx_batches_created_at,sort:desc;index:idx_batches_status_created_at,priority:2,sort:desc"`
	UpdatedAt      time.Time         `gorm:"column:updated_at;not null"`
	AnchorProof    *AnchorProofModel `gorm:"foreignKey:BatchID;references:BatchID"`
}

func (BatchModel) TableName() string { return "batches" }

type AnchorProofModel struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	BatchID         string    `gorm:"column:batch_id;type:text;not null;uniqueIndex"`
	TxHash          string    `gorm:"column:tx_hash;type:text;not null;index:idx_anchor_proofs_tx_hash"`
	BlockNumber     int64     `gorm:"column:block_number;not null"`
	ChainID         string    `gorm:"column:chain_id;type:text;not null"`
	ContractAddress string    `gorm:"column:contract_address;type:text;not null"`
	AnchorHash      string    `gorm:"column:anchor_hash;type:text;not null"`
	AnchoredAt      time.Time `gorm:"column:anchored_at;not null"`
}

func (AnchorProofModel) TableName() string { return "anchor_proofs" }

type ReconcileJobModel struct {
	ID             uint      `gorm:"primaryKey;autoIncrement"`
	JobID          string    `gorm:"column:job_id;type:text;not null;uniqueIndex"`
	TriggerType    string    `gorm:"column:trigger_type;type:text;not null"`
	Status         string    `gorm:"column:status;type:text;not null"`
	RequestedCount int       `gorm:"column:requested_count;not null"`
	ScheduledCount int       `gorm:"column:scheduled_count;not null"`
	SkippedCount   int       `gorm:"column:skipped_count;not null"`
	ErrorMessage   *string   `gorm:"column:error_message;type:text"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (ReconcileJobModel) TableName() string { return "reconcile_jobs" }

type ReconcileJobItemModel struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	JobID        string    `gorm:"column:job_id;type:text;not null;uniqueIndex:uq_reconcile_job_item,priority:1"`
	BatchID      string    `gorm:"column:batch_id;type:text;not null;uniqueIndex:uq_reconcile_job_item,priority:2"`
	BeforeStatus string    `gorm:"column:before_status;type:text;not null"`
	AfterStatus  string    `gorm:"column:after_status;type:text;not null"`
	AttemptNo    int       `gorm:"column:attempt_no;not null"`
	ErrorMessage *string   `gorm:"column:error_message;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

func (ReconcileJobItemModel) TableName() string { return "reconcile_job_items" }

type AuditLogModel struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	EventType   string    `gorm:"column:event_type;type:text;not null;index:idx_audit_logs_event_time,priority:1"`
	EntityType  string    `gorm:"column:entity_type;type:text;not null;index:idx_audit_logs_entity,priority:1"`
	EntityID    string    `gorm:"column:entity_id;type:text;not null;index:idx_audit_logs_entity,priority:2"`
	Status      *string   `gorm:"column:status;type:text"`
	Message     *string   `gorm:"column:message;type:text"`
	RequestID   *string   `gorm:"column:request_id;type:text"`
	PayloadJSON *string   `gorm:"column:payload_json;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;index:idx_audit_logs_entity,priority:3,sort:desc;index:idx_audit_logs_event_time,priority:2,sort:desc"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }
