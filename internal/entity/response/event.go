package response

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"time"
)

type EventCreationResponse struct {
	EventID uuid.UUID `json:"event_id"`
}

type EventSimpleResponse struct {
	ID        uuid.UUID         `json:"id"`
	Name      string            `json:"name"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Status    model.EventStatus `json:"status"`
}

type EventDetailedResponse struct {
	EventSimpleResponse
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	CalculateNodePool  bool                `json:"calculate_node_pool"`
	Cluster            Cluster             `json:"cluster"`
	ModifiedHPAConfigs []ModifiedHPAConfig `json:"modified_hpa_configs"`
	UpdatedNodePools   []UpdatedNodePool   `json:"updated_node_pools"`
}
