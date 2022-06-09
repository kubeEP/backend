package model

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"
)

type Cluster struct {
	BaseModel
	DatacenterID        gormDatatype.UUID   `json:"datacenter_id"`
	Metadata            gormDatatype.JSON   `json:"metadata"`
	Name                string              `json:"name"`
	Certificate         string              `json:"certificate"`
	ServerEndpoint      string              `json:"server_endpoint"`
	Datacenter          Datacenter          `gorm:"ForeignKey:DatacenterID;constraint:OnDelete:CASCADE" json:"-"`
	LatestHPAAPIVersion constant.HPAVersion `json:"latest_hpa_api_version" gorm:"column:latest_hpa_api_version"`
}

func (c *Cluster) TableName() string {
	return "clusters"
}

type ClusterWithDatacenterType struct {
}
