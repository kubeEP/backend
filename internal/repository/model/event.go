package model

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"
	"time"
)

type EventStatus string

const (
	EventFailed    EventStatus = "FAILED"
	EventSuccess   EventStatus = "SUCCESS"
	EventExecuting EventStatus = "EXECUTING"
	EventPrescaled EventStatus = "PRESCALED"
	EventWatching  EventStatus = "WATCHING"
	EventPending   EventStatus = "PENDING"
)

type Event struct {
	BaseModel
	Name              string
	StartTime         time.Time
	EndTime           time.Time
	ClusterID         gormDatatype.UUID
	Status            EventStatus `gorm:"default:PENDING"`
	Message           string
	CalculateNodePool bool
	Cluster           Cluster `gorm:"ForeignKey:ClusterID;constraint:OnDelete:CASCADE"`
}

func (e *Event) TableName() string {
	return "events"
}
