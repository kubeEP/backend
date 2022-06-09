package repository

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
	"time"
)

type Event interface {
	GetEventByID(tx *gorm.DB, id uuid.UUID) (*model.Event, error)
	GetEventByName(tx *gorm.DB, name string) (*model.Event, error)
	ListEventByClusterID(tx *gorm.DB, id uuid.UUID) ([]*model.Event, error)
	InsertEvent(tx *gorm.DB, data *model.Event) error
	SaveEvent(tx *gorm.DB, data *model.Event) error
	DeleteEvent(tx *gorm.DB, id uuid.UUID) error
	FindEventByStatusWithStarTimeBeforeMinuteAndClusterData(
		tx *gorm.DB,
		status model.EventStatus,
		minute int,
		now time.Time,
	) (
		[]*model.Event,
		error,
	)
	FindEventByStatusWithStarTimeBeforeMinute(
		tx *gorm.DB,
		status model.EventStatus,
		minute int,
		now time.Time,
	) (
		[]*model.Event,
		error,
	)
	FindWatchedEvent(tx *gorm.DB, now time.Time) ([]*model.Event, error)
	FinishWatchedEvent(tx *gorm.DB, now time.Time) error
}

type event struct {
}

func newEvent() Event {
	return &event{}
}

func (e *event) GetEventByID(tx *gorm.DB, id uuid.UUID) (*model.Event, error) {
	data := &model.Event{}
	tx = tx.Model(data).First(data, id)
	return data, tx.Error
}

func (e *event) GetEventByName(tx *gorm.DB, name string) (*model.Event, error) {
	data := &model.Event{}
	tx = tx.Model(data).Where("name = ?", name).First(data)
	return data, tx.Error
}

func (e *event) ListEventByClusterID(tx *gorm.DB, id uuid.UUID) ([]*model.Event, error) {
	var data []*model.Event
	tx = tx.Model(&model.Event{}).Where("cluster_id = ?", id).Find(&data)
	return data, tx.Error
}

func (e *event) InsertEvent(tx *gorm.DB, data *model.Event) error {
	return tx.Create(data).Error
}

func (e *event) SaveEvent(tx *gorm.DB, data *model.Event) error {
	return tx.Save(data).Error
}

func (e *event) DeleteEvent(tx *gorm.DB, id uuid.UUID) error {
	return tx.Delete(&model.Event{}, "id = ?", id).Error
}

func (e *event) FindWatchedEvent(tx *gorm.DB, now time.Time) ([]*model.Event, error) {
	var data []*model.Event
	tx = tx.Model(&model.Event{}).Where(
		"status = ? and end_time < ?", model.EventWatching, now.UTC(),
	).Find(&data)
	return data, tx.Error
}

func (e *event) FinishWatchedEvent(tx *gorm.DB, now time.Time) error {
	return tx.Model(&model.Event{}).Where(
		"status = ? and end_time < ?", model.EventWatching, now.UTC(),
	).Update("status", model.EventSuccess).Error
}

func (e *event) FindEventByStatusWithStarTimeBeforeMinuteAndClusterData(
	tx *gorm.DB,
	status model.EventStatus,
	minute int,
	now time.Time,
) (
	[]*model.Event,
	error,
) {
	var data []*model.Event
	rows, err := tx.Raw(
		`select 
    e.id, 
    e.created_at, 
    e.updated_at, 
    e.deleted_at, 
    e.name, 
    e.start_time, 
    e.end_time, 
    e.cluster_id, 
    e.status, 
    e.message,
    c.name, 
    d.datacenter,
    e.calculate_node_pool from events e 
    join clusters c on c.id = e.cluster_id and c.deleted_at is null
    join datacenters d on d.id = c.datacenter_id and d.deleted_at is null
             where e.start_time - ? < ? * interval '1 minutes' and e.status = ? and e.deleted_at is null`,
		now.UTC(),
		minute+1,
		status,
	).Rows()
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		eventData := &model.Event{}
		err = rows.Scan(
			&eventData.ID,
			&eventData.CreatedAt,
			&eventData.UpdatedAt,
			&eventData.DeletedAt,
			&eventData.Name,
			&eventData.StartTime,
			&eventData.EndTime,
			&eventData.ClusterID,
			&eventData.Status,
			&eventData.Message,
			&eventData.Cluster.Name,
			&eventData.Cluster.Datacenter.Datacenter,
			&eventData.CalculateNodePool,
		)
		if err != nil {
			return nil, err
		}
		data = append(data, eventData)
	}
	return data, nil
}

func (e *event) FindEventByStatusWithStarTimeBeforeMinute(
	tx *gorm.DB,
	status model.EventStatus,
	minute int,
	now time.Time,
) (
	[]*model.Event,
	error,
) {
	var data []*model.Event
	tx = tx.Model(&model.Event{}).
		Where(
			"start_time - ? < ? * interval '1 minutes' and status = ?",
			now.UTC(),
			minute+1,
			status,
		).Find(&data)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return data, nil
}
