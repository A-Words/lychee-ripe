package domain

import "time"

type OrchardRecord struct {
	OrchardID   string
	OrchardName string
	Status      ResourceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PlotRecord struct {
	PlotID      string
	OrchardID   string
	PlotName    string
	Status      ResourceStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
