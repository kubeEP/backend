package model

import gormDatatype "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"

type UpdatedNodePool struct {
	BaseModel
	NodePoolName string
	MaxNode      int32
	EventID      gormDatatype.UUID
	Event        Event `gorm:"ForeignKey:EventID;constraint:OnDelete:CASCADE"`
}

func (UpdatedNodePool) TableName() string {
	return "updated_node_pool"
}
