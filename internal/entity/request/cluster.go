package request

import "github.com/google/uuid"

type ExistingClusterData struct {
	ClusterID *uuid.UUID `json:"cluster_id" query:"cluster_id" validate:"required"`
}
