package UCEntity

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"time"
)

type Event struct {
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ID                uuid.UUID
	Name              string
	StartTime         time.Time
	EndTime           time.Time
	Status            model.EventStatus
	Message           string
	CalculateNodePool bool
	Cluster           ClusterData
}

type DetailedEvent struct {
	Event
	EventModifiedHPAConfigData []EventModifiedHPAConfigData
}
