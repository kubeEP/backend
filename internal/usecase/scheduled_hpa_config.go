package useCase

import (
	"github.com/google/uuid"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
)

type ScheduledHPAConfig interface {
	RegisterModifiedHPAConfigs(
		tx *gorm.DB,
		modifiedHPAs []UCEntity.EventModifiedHPAConfigData,
		eventID uuid.UUID,
	) ([]uuid.UUID, error)
	DeleteEventModifiedHPAConfigs(tx *gorm.DB, eventID uuid.UUID) error
	SoftDeleteEventModifiedHPAConfigs(
		tx *gorm.DB,
		eventID uuid.UUID,
	) error
	ListScheduledHPAConfigByEventID(
		tx *gorm.DB,
		eventID uuid.UUID,
	) ([]*UCEntity.EventModifiedHPAConfigData, error)
	UpdateScheduledHPAConfigStatusMessage(
		tx *gorm.DB,
		id uuid.UUID,
		status model.HPAUpdateStatus,
		msg string,
	) error
}

type scheduledHPAConfig struct {
	scheduledHPAConfigRepo repository.ScheduledHPAConfig
}

func newScheduledHPAConfig(scheduledHPAConfigRepo repository.ScheduledHPAConfig) ScheduledHPAConfig {
	return &scheduledHPAConfig{scheduledHPAConfigRepo: scheduledHPAConfigRepo}
}

func (s *scheduledHPAConfig) RegisterModifiedHPAConfigs(
	tx *gorm.DB,
	modifiedHPAs []UCEntity.EventModifiedHPAConfigData,
	eventID uuid.UUID,
) ([]uuid.UUID, error) {
	var data []*model.ScheduledHPAConfig
	for _, modifiedHPA := range modifiedHPAs {
		modelData := &model.ScheduledHPAConfig{
			Name:      modifiedHPA.Name,
			MinPods:   modifiedHPA.MinReplicas,
			MaxPods:   modifiedHPA.MaxReplicas,
			Namespace: modifiedHPA.Namespace,
		}
		modelData.EventID.SetUUID(eventID)
		data = append(
			data, modelData,
		)
	}
	err := s.scheduledHPAConfigRepo.InsertBatchScheduledHPAConfig(tx, data)
	if err != nil {
		return nil, err
	}
	var uuids []uuid.UUID
	for _, datum := range data {
		uuids = append(uuids, datum.ID.GetUUID())
	}
	return uuids, nil
}

func (s *scheduledHPAConfig) DeleteEventModifiedHPAConfigs(tx *gorm.DB, eventID uuid.UUID) error {
	return s.scheduledHPAConfigRepo.DeletePermanentAllHPAConfigByEventID(tx, eventID)
}

func (s *scheduledHPAConfig) ListScheduledHPAConfigByEventID(
	tx *gorm.DB,
	eventID uuid.UUID,
) ([]*UCEntity.EventModifiedHPAConfigData, error) {
	scheduledHPAConfigs, err := s.scheduledHPAConfigRepo.ListScheduledHPAConfigByEventID(
		tx,
		eventID,
	)
	if err != nil {
		return nil, err
	}

	var eventModifiedHPAConfigData []*UCEntity.EventModifiedHPAConfigData
	for _, hpa := range scheduledHPAConfigs {
		eventModifiedHPAConfigData = append(
			eventModifiedHPAConfigData, &UCEntity.EventModifiedHPAConfigData{
				ID:          hpa.ID.GetUUID(),
				Name:        hpa.Name,
				Status:      hpa.Status,
				Message:     hpa.Message,
				Namespace:   hpa.Namespace,
				MinReplicas: hpa.MinPods,
				MaxReplicas: hpa.MaxPods,
			},
		)
	}

	return eventModifiedHPAConfigData, nil
}

func (s *scheduledHPAConfig) SoftDeleteEventModifiedHPAConfigs(
	tx *gorm.DB,
	eventID uuid.UUID,
) error {
	return s.scheduledHPAConfigRepo.DeleteAllHPAConfigByEventID(tx, eventID)
}

func (s *scheduledHPAConfig) UpdateScheduledHPAConfigStatusMessage(
	tx *gorm.DB,
	id uuid.UUID,
	status model.HPAUpdateStatus,
	msg string,
) error {
	scheduledHPAConfigData, err := s.scheduledHPAConfigRepo.GetScheduledHPAConfigByID(tx, id)
	if err != nil {
		return err
	}

	scheduledHPAConfigData.Status = status
	scheduledHPAConfigData.Message = msg

	return s.scheduledHPAConfigRepo.SaveScheduledHPAConfig(tx, scheduledHPAConfigData)
}
