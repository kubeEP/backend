package response

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
)

type Cluster struct {
	ID             *uuid.UUID               `json:"id,omitempty"`
	Name           string                   `json:"name"`
	Datacenter     model.DatacenterProvider `json:"datacenter"`
	DatacenterName string                   `json:"datacenter_name,omitempty"`
}

type UpdatedNodePool struct {
	ID           uuid.UUID `json:"id"`
	NodePoolName string    `json:"node_pool_name"`
	MaxNode      int32     `json:"max_node"`
}

type ClusterDetailResponse struct {
	Cluster Cluster     `json:"cluster"`
	HPAList []SimpleHPA `json:"hpa_list"`
}
