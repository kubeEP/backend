package model

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"
)

type DatacenterProvider string

const (
	GCP DatacenterProvider = "GCP"
)

type Datacenter struct {
	BaseModel
	Name        string             `json:"name"`
	Credentials gormDatatype.JSON  `json:"credentials"`
	Metadata    gormDatatype.JSON  `json:"metadata"`
	Datacenter  DatacenterProvider `json:"datacenter"`
}

func (d *Datacenter) TableName() string {
	return "datacenters"
}
