package repository

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
)

type ScheduledHPAConfig interface {
	GetScheduledHPAConfigByID(tx *gorm.DB, id uuid.UUID) (*model.ScheduledHPAConfig, error)
	ListScheduledHPAConfigByEventID(tx *gorm.DB, id uuid.UUID) ([]*model.ScheduledHPAConfig, error)
	InsertScheduledHPAConfig(tx *gorm.DB, data *model.ScheduledHPAConfig) error
	InsertBatchScheduledHPAConfig(
		tx *gorm.DB,
		data []*model.ScheduledHPAConfig,
	) error
	DeletePermanentAllHPAConfigByEventID(
		tx *gorm.DB,
		eventID uuid.UUID,
	) error
	DeleteAllHPAConfigByEventID(tx *gorm.DB, eventID uuid.UUID) error
	SaveScheduledHPAConfig(
		tx *gorm.DB,
		data *model.ScheduledHPAConfig,
	) error
}

type scheduledHPAConfig struct {
}

func newScheduledHPAConfig() ScheduledHPAConfig {
	return &scheduledHPAConfig{}
}

func (s *scheduledHPAConfig) GetScheduledHPAConfigByID(
	tx *gorm.DB,
	id uuid.UUID,
) (*model.ScheduledHPAConfig, error) {
	data := &model.ScheduledHPAConfig{}
	tx = tx.Model(data).First(data, id)
	return data, tx.Error
}

func (s *scheduledHPAConfig) ListScheduledHPAConfigByEventID(
	tx *gorm.DB,
	id uuid.UUID,
) ([]*model.ScheduledHPAConfig, error) {
	var data []*model.ScheduledHPAConfig
	tx = tx.Model(&model.ScheduledHPAConfig{}).Where("event_id = ?", id).Find(&data)
	return data, tx.Error
}

func (s *scheduledHPAConfig) InsertScheduledHPAConfig(
	tx *gorm.DB,
	data *model.ScheduledHPAConfig,
) error {
	return tx.Create(data).Error
}

func (s *scheduledHPAConfig) InsertBatchScheduledHPAConfig(
	tx *gorm.DB,
	data []*model.ScheduledHPAConfig,
) error {
	return tx.Create(data).Error
}

func (s *scheduledHPAConfig) DeletePermanentAllHPAConfigByEventID(
	tx *gorm.DB,
	eventID uuid.UUID,
) error {
	return tx.Unscoped().Where("event_id = ?", eventID).Delete(&model.ScheduledHPAConfig{}).Error
}

func (s *scheduledHPAConfig) DeleteAllHPAConfigByEventID(tx *gorm.DB, eventID uuid.UUID) error {
	return tx.Delete(&model.ScheduledHPAConfig{}, "event_id = ?", eventID).Error
}

func (s *scheduledHPAConfig) SaveScheduledHPAConfig(
	tx *gorm.DB,
	data *model.ScheduledHPAConfig,
) error {
	return tx.Save(data).Error
}
