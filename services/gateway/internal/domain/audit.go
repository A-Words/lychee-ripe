package domain

import "time"

type AuditLogRecord struct {
	EventType   string
	EntityType  string
	EntityID    string
	Status      *string
	Message     *string
	RequestID   *string
	PayloadJSON *string
	CreatedAt   time.Time
}
