package request

import "github.com/google/uuid"

type GCPRegisterClusterData struct {
	ClustersName          []string   `json:"clusters_name" validate:"required"`
	DatacenterID          *uuid.UUID `json:"datacenter_id" validate:"required"`
	IsDatacenterTemporary *bool      `json:"is_datacenter_temporary" validate:"required"`
}
