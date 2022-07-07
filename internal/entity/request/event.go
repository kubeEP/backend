package request

import (
	"github.com/google/uuid"
	"time"
)

type EventDataRequest struct {
	Name               *string                      `json:"name" validate:"required"`
	StartTime          *time.Time                   `json:"start_time" validate:"required,gtefield=ExecuteConfigAt"`
	EndTime            *time.Time                   `json:"end_time" validate:"required,gtefield=StartTime"`
	ClusterID          *uuid.UUID                   `json:"cluster_id" validate:"required"`
	CalculateNodePool  *bool                        `json:"calculate_node_pool"`
	ExecuteConfigAt    *time.Time                   `json:"execute_config_at" validate:"required"`
	WatchingAt         *time.Time                   `json:"watching_at" validate:"required,gtefield=ExecuteConfigAt,ltefield=StartTime"`
	ModifiedHPAConfigs []EventModifiedHPAConfigData `json:"modified_hpa_configs" validate:"required,min=1,dive"`
}

type EventListRequest struct {
	ClusterID *uuid.UUID `query:"cluster_id" validate:"required"`
}

type UpdateEventDataRequest struct {
	Name               *string                      `json:"name" validate:"required"`
	StartTime          *time.Time                   `json:"start_time" validate:"required"`
	EndTime            *time.Time                   `json:"end_time" validate:"required,gtefield=StartTime"`
	ModifiedHPAConfigs []EventModifiedHPAConfigData `json:"modified_hpa_configs" validate:"required,min=1,dive"`
	CalculateNodePool  *bool                        `json:"calculate_node_pool"`
	ExecuteConfigAt    *time.Time                   `json:"execute_config_at" validate:"required,gtefield=ExecuteConfigAt"`
	WatchingAt         *time.Time                   `json:"watching_at" validate:"required,gtefield=ExecuteConfigAt,ltefield=StartTime"`
	EventID            *uuid.UUID                   `json:"event_id" validator:"required"`
}

type EventDetailRequest struct {
	EventID *uuid.UUID `json:"event_id" query:"event_id" validator:"required"`
}
