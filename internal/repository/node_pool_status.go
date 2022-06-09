package repository

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
)

type NodePoolStatus interface {
	GetAllNodePoolStatusByUpdatedNodePoolID(
		tx *gorm.DB,
		updatedNodePoolID uuid.UUID,
	) ([]*model.NodePoolStatus, error)
	GetAllNodePoolStatusByEventID(
		tx *gorm.DB,
		eventID uuid.UUID,
	) ([]*model.NodePoolStatus, error)
}

type nodePoolStatus struct {
}

func newNodePoolStatus() NodePoolStatus {
	return &nodePoolStatus{}
}

func (n *nodePoolStatus) GetAllNodePoolStatusByEventID(
	tx *gorm.DB,
	eventID uuid.UUID,
) ([]*model.NodePoolStatus, error) {
	var output []*model.NodePoolStatus
	err := tx.Table("node_pool_status n").Joins("updated_node_pool u on u.id = n.updated_node_pool_id and u.deleted_at is null").Where(
		"u.event_id = ?",
		eventID,
	).Find(&output).Error
	return output, err
}

func (n *nodePoolStatus) GetAllNodePoolStatusByUpdatedNodePoolID(
	tx *gorm.DB,
	updatedNodePoolID uuid.UUID,
) ([]*model.NodePoolStatus, error) {
	var output []*model.NodePoolStatus
	err := tx.Model(&model.NodePoolStatus{}).Where(
		"updated_node_pool_id = ?",
		updatedNodePoolID,
	).Find(&output).Error
	return output, err
}
