package useCase

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
	"time"
)

type Event interface {
	RegisterEvents(tx *gorm.DB, eventData *UCEntity.Event) (uuid.UUID, error)
	GetEventByName(tx *gorm.DB, eventName string) (*UCEntity.Event, error)
	ListEventByClusterID(tx *gorm.DB, clusterID uuid.UUID) ([]UCEntity.Event, error)
	UpdateEvent(tx *gorm.DB, eventData *UCEntity.Event) error
	GetEventByID(tx *gorm.DB, eventID uuid.UUID) (*UCEntity.Event, error)
	GetDetailedEventData(tx *gorm.DB, eventID uuid.UUID) (
		*UCEntity.DetailedEvent,
		error,
	)
	DeleteEvent(tx *gorm.DB, id uuid.UUID) error
	GetAllPendingExecutableEvent(tx *gorm.DB, now time.Time) (
		[]*UCEntity.Event,
		error,
	)
	GetAllPrescaledEvent10MinBeforeStart(tx *gorm.DB, now time.Time) (
		[]*UCEntity.Event,
		error,
	)
	FinishAllWatchedEvent(tx *gorm.DB, now time.Time) error
}

type event struct {
	validatorInst                *validator.Validate
	eventRepository              repository.Event
	scheduledHPAConfigRepository repository.ScheduledHPAConfig
	clusterRepository            repository.Cluster
}

func newEvent(
	validatorInst *validator.Validate,
	eventRepository repository.Event,
	scheduledHPAConfigRepository repository.ScheduledHPAConfig,
	clusterRepository repository.Cluster,
) Event {
	return &event{
		validatorInst:                validatorInst,
		eventRepository:              eventRepository,
		scheduledHPAConfigRepository: scheduledHPAConfigRepository,
		clusterRepository:            clusterRepository,
	}
}

func (e *event) RegisterEvents(tx *gorm.DB, eventData *UCEntity.Event) (uuid.UUID, error) {
	data := &model.Event{
		Name:              eventData.Name,
		StartTime:         eventData.StartTime,
		EndTime:           eventData.EndTime,
		CalculateNodePool: eventData.CalculateNodePool,
	}
	data.ClusterID.SetUUID(eventData.Cluster.ID)

	err := e.eventRepository.InsertEvent(tx, data)
	if err != nil {
		return uuid.UUID{}, err
	}
	return data.ID.GetUUID(), nil
}

func (e *event) GetEventByName(tx *gorm.DB, eventName string) (*UCEntity.Event, error) {
	data, err := e.eventRepository.GetEventByName(tx, eventName)
	if err != nil {
		return nil, err
	}
	return &UCEntity.Event{
		ID:                data.ID.GetUUID(),
		Name:              data.Name,
		StartTime:         data.StartTime,
		EndTime:           data.EndTime,
		CreatedAt:         data.CreatedAt,
		UpdatedAt:         data.UpdatedAt,
		Status:            data.Status,
		Message:           data.Message,
		CalculateNodePool: data.CalculateNodePool,
	}, nil
}

func (e *event) GetEventByID(tx *gorm.DB, eventID uuid.UUID) (*UCEntity.Event, error) {
	data, err := e.eventRepository.GetEventByID(tx, eventID)
	if err != nil {
		return nil, err
	}
	return &UCEntity.Event{
		ID:                eventID,
		Name:              data.Name,
		StartTime:         data.StartTime,
		EndTime:           data.EndTime,
		CreatedAt:         data.CreatedAt,
		UpdatedAt:         data.UpdatedAt,
		Status:            data.Status,
		Message:           data.Message,
		CalculateNodePool: data.CalculateNodePool,
		Cluster:           UCEntity.ClusterData{ID: data.ClusterID.GetUUID()},
	}, nil
}

func (e *event) ListEventByClusterID(tx *gorm.DB, clusterID uuid.UUID) ([]UCEntity.Event, error) {
	events, err := e.eventRepository.ListEventByClusterID(tx, clusterID)
	if err != nil {
		return nil, err
	}
	var output []UCEntity.Event
	for _, event := range events {
		output = append(
			output, UCEntity.Event{
				ID:                event.ID.GetUUID(),
				Name:              event.Name,
				StartTime:         event.StartTime,
				EndTime:           event.EndTime,
				CreatedAt:         event.CreatedAt,
				UpdatedAt:         event.UpdatedAt,
				Status:            event.Status,
				Message:           event.Message,
				CalculateNodePool: event.CalculateNodePool,
			},
		)
	}
	return output, nil
}

func (e *event) UpdateEvent(tx *gorm.DB, eventData *UCEntity.Event) error {
	data := &model.Event{
		Name:              eventData.Name,
		StartTime:         eventData.StartTime,
		EndTime:           eventData.EndTime,
		Status:            eventData.Status,
		Message:           eventData.Message,
		CalculateNodePool: eventData.CalculateNodePool,
	}
	data.CreatedAt = eventData.CreatedAt
	data.UpdatedAt = eventData.UpdatedAt
	data.ID.SetUUID(eventData.ID)
	data.ClusterID.SetUUID(eventData.Cluster.ID)
	return e.eventRepository.SaveEvent(tx, data)
}

func (e *event) GetDetailedEventData(tx *gorm.DB, eventID uuid.UUID) (
	*UCEntity.DetailedEvent,
	error,
) {
	eventData, err := e.eventRepository.GetEventByID(tx, eventID)
	if err != nil {
		return nil, err
	}

	clusterData, err := e.clusterRepository.GetClusterWithDatacenterByID(
		tx,
		eventData.ClusterID.GetUUID(),
	)
	if err != nil {
		return nil, err
	}

	scheduledHPAConfigs, err := e.scheduledHPAConfigRepository.ListScheduledHPAConfigByEventID(
		tx,
		eventData.ID.GetUUID(),
	)
	if err != nil {
		return nil, err
	}

	data := &UCEntity.DetailedEvent{
		Event: UCEntity.Event{
			CreatedAt:         eventData.CreatedAt,
			UpdatedAt:         eventData.UpdatedAt,
			ID:                eventID,
			Name:              eventData.Name,
			StartTime:         eventData.StartTime,
			Status:            eventData.Status,
			Message:           eventData.Message,
			EndTime:           eventData.EndTime,
			CalculateNodePool: eventData.CalculateNodePool,
			Cluster: UCEntity.ClusterData{
				ID:   eventData.ClusterID.GetUUID(),
				Name: clusterData.Name,
				Datacenter: UCEntity.DatacenterDetailedData{
					Datacenter: clusterData.Datacenter.Datacenter,
					Name:       clusterData.Datacenter.Name,
				},
			},
		},
	}

	var eventModifiedHPAConfigData []UCEntity.EventModifiedHPAConfigData
	for _, hpa := range scheduledHPAConfigs {
		eventModifiedHPAConfigData = append(
			eventModifiedHPAConfigData, UCEntity.EventModifiedHPAConfigData{
				ID:          hpa.ID.GetUUID(),
				Name:        hpa.Name,
				Namespace:   hpa.Namespace,
				Status:      hpa.Status,
				Message:     hpa.Message,
				MinReplicas: hpa.MinPods,
				MaxReplicas: hpa.MaxPods,
			},
		)
	}

	data.EventModifiedHPAConfigData = eventModifiedHPAConfigData
	return data, nil
}

func (e *event) DeleteEvent(tx *gorm.DB, id uuid.UUID) error {
	return e.eventRepository.DeleteEvent(tx, id)
}

func (e *event) GetAllPendingExecutableEvent(tx *gorm.DB, now time.Time) (
	[]*UCEntity.Event,
	error,
) {
	events, err := e.eventRepository.FindEventByStatusWithStarTimeBeforeMinuteAndClusterData(
		tx,
		model.EventPending,
		60,
		now,
	)
	if err != nil {
		return nil, err
	}
	var eventsData []*UCEntity.Event
	for _, event := range events {
		eventsData = append(
			eventsData, &UCEntity.Event{
				CreatedAt:         event.CreatedAt,
				UpdatedAt:         event.UpdatedAt,
				ID:                event.ID.GetUUID(),
				Status:            event.Status,
				Name:              event.Name,
				Message:           event.Message,
				StartTime:         event.StartTime,
				EndTime:           event.EndTime,
				CalculateNodePool: event.CalculateNodePool,
				Cluster:           UCEntity.ClusterData{Name: event.Cluster.Name, ID: event.ClusterID.GetUUID(), Datacenter: UCEntity.DatacenterDetailedData{Datacenter: event.Cluster.Datacenter.Datacenter}},
			},
		)
	}

	return eventsData, nil
}

func (e *event) GetAllPrescaledEvent10MinBeforeStart(tx *gorm.DB, now time.Time) (
	[]*UCEntity.Event,
	error,
) {
	events, err := e.eventRepository.FindEventByStatusWithStarTimeBeforeMinute(
		tx,
		model.EventPrescaled,
		55,
		now,
	)
	if err != nil {
		return nil, err
	}
	var eventsData []*UCEntity.Event
	for _, event := range events {
		eventsData = append(
			eventsData, &UCEntity.Event{
				CreatedAt:         event.CreatedAt,
				UpdatedAt:         event.UpdatedAt,
				ID:                event.ID.GetUUID(),
				Status:            event.Status,
				Name:              event.Name,
				Message:           event.Message,
				StartTime:         event.StartTime,
				EndTime:           event.EndTime,
				CalculateNodePool: event.CalculateNodePool,
				Cluster:           UCEntity.ClusterData{Name: event.Cluster.Name, ID: event.ClusterID.GetUUID(), Datacenter: UCEntity.DatacenterDetailedData{Datacenter: event.Cluster.Datacenter.Datacenter}},
			},
		)
	}

	return eventsData, nil
}

func (e *event) FinishAllWatchedEvent(tx *gorm.DB, now time.Time) error {
	return e.eventRepository.FinishWatchedEvent(tx, now)
}
