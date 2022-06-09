package UCEntity

import (
	"github.com/google/uuid"
	"time"
)

type UpdatedNodePoolData struct {
	ID           uuid.UUID
	NodePoolName string
	MaxNode      int32
}

type NodePoolStatusData struct {
	CreatedAt time.Time
	Count     int32
}

type HPAStatusData struct {
	CreatedAt           time.Time
	Replicas            int32
	AvailableReplicas   int32
	ReadyReplicas       int32
	UnavailableReplicas int32
}
