package useCase

import (
	"github.com/google/uuid"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"gorm.io/gorm"
)

type Statistic interface {
	GetAllUpdatedNodePoolByEvent(
		tx *gorm.DB,
		eventID uuid.UUID,
	) ([]*UCEntity.UpdatedNodePoolData, error)
	GetAllNodePoolStatusByUpdatedNodePoolID(
		tx *gorm.DB,
		updatedNodePoolID uuid.UUID,
	) ([]*UCEntity.NodePoolStatusData, error)
	GetAllHPAStatusByScheduledHPAConfigID(
		tx *gorm.DB,
		scheduledHPAConfigID uuid.UUID,
	) ([]*UCEntity.HPAStatusData, error)
}

type statistic struct {
	updatedNodePoolRepo repository.UpdatedNodePool
	hpaStatusRepo       repository.HPAStatus
	nodePoolStatusRepo  repository.NodePoolStatus
}

func newStatistic(
	updatedNodePoolRepo repository.UpdatedNodePool,
	hpaStatusRepo repository.HPAStatus,
	nodePoolStatusRepo repository.NodePoolStatus,
) Statistic {
	return &statistic{
		updatedNodePoolRepo: updatedNodePoolRepo,
		hpaStatusRepo:       hpaStatusRepo,
		nodePoolStatusRepo:  nodePoolStatusRepo,
	}
}

func (u *statistic) GetAllUpdatedNodePoolByEvent(
	tx *gorm.DB,
	eventID uuid.UUID,
) ([]*UCEntity.UpdatedNodePoolData, error) {
	var output []*UCEntity.UpdatedNodePoolData
	data, err := u.updatedNodePoolRepo.GetAllUpdatedNodePoolByEventID(tx, eventID)
	if err != nil {
		return nil, err
	}
	for _, d := range data {
		output = append(
			output, &UCEntity.UpdatedNodePoolData{
				ID:           d.ID.GetUUID(),
				NodePoolName: d.NodePoolName,
				MaxNode:      d.MaxNode,
			},
		)
	}
	return output, nil
}

func (u *statistic) GetAllNodePoolStatusByUpdatedNodePoolID(
	tx *gorm.DB,
	updatedNodePoolID uuid.UUID,
) ([]*UCEntity.NodePoolStatusData, error) {
	var output []*UCEntity.NodePoolStatusData
	data, err := u.nodePoolStatusRepo.GetAllNodePoolStatusByUpdatedNodePoolID(tx, updatedNodePoolID)
	if err != nil {
		return nil, err
	}
	for _, d := range data {
		output = append(
			output, &UCEntity.NodePoolStatusData{
				CreatedAt: d.CreatedAt,
				Count:     d.NodeCount,
			},
		)
	}
	return output, nil
}

func (u *statistic) GetAllHPAStatusByScheduledHPAConfigID(
	tx *gorm.DB,
	scheduledHPAConfigID uuid.UUID,
) ([]*UCEntity.HPAStatusData, error) {
	var output []*UCEntity.HPAStatusData
	data, err := u.hpaStatusRepo.GetAllHPAStatusByScheduledHPAConfigID(
		tx,
		scheduledHPAConfigID,
	)
	if err != nil {
		return nil, err
	}
	for _, d := range data {
		output = append(
			output, &UCEntity.HPAStatusData{
				CreatedAt:           d.CreatedAt,
				Replicas:            d.Replicas,
				ReadyReplicas:       d.ReadyReplicas,
				AvailableReplicas:   d.AvailableReplicas,
				UnavailableReplicas: d.UnavailableReplicas,
			},
		)
	}
	return output, nil
}
