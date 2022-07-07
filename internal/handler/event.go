package handler

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/request"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/response"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
	"gorm.io/gorm"
	"time"
)

type Event interface {
	RegisterEvents(c *fiber.Ctx) error
	ListEventByCluster(c *fiber.Ctx) error
	UpdateEvent(c *fiber.Ctx) error
	GetDetailedEvent(c *fiber.Ctx) error
	DeleteEvent(c *fiber.Ctx) error
	ListNodePoolStatusByUpdatedNodePool(c *fiber.Ctx) error
	ListHPAStatusByScheduledHPAConfig(c *fiber.Ctx) error
}

type event struct {
	kubernetesBaseHandler
	validatorInst        *validator.Validate
	db                   *gorm.DB
	eventUC              useCase.Event
	scheduledHPAConfigUC useCase.ScheduledHPAConfig
	statisticUC          useCase.Statistic
}

func newEventHandler(
	validatorInst *validator.Validate,
	eventUC useCase.Event,
	scheduledHPAConfigUC useCase.ScheduledHPAConfig,
	updatedNodePoolUC useCase.Statistic,
	db *gorm.DB,
	kubeHandler kubernetesBaseHandler,
) Event {
	return &event{
		kubernetesBaseHandler: kubeHandler,
		validatorInst:         validatorInst,
		eventUC:               eventUC,
		scheduledHPAConfigUC:  scheduledHPAConfigUC,
		statisticUC:           updatedNodePoolUC,
		db:                    db,
	}
}

func (e *event) RegisterEvents(c *fiber.Ctx) error {
	reqData := &request.EventDataRequest{}

	err := c.BodyParser(reqData)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}
	err = e.validatorInst.Struct(reqData)
	if err != nil {
		return e.errorResponse(c, errorConstant.InvalidRequestBody)
	}

	if reqData.CalculateNodePool == nil {
		active := true
		reqData.CalculateNodePool = &active
	}

	utcNow := time.Now().UTC()
	if utcNow.After(*reqData.StartTime) || utcNow.After(*reqData.EndTime) {
		return e.errorResponse(c, errorConstant.InvalidRequestBody)
	}

	ctx := c.Context()
	db := e.db.WithContext(ctx)
	tx := db.Begin()

	kubernetesClient, clusterData, err := e.getClusterKubernetesClient(
		ctx,
		db,
		*reqData.ClusterID,
	)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	HPAs, err := e.generalClusterUC.GetAllHPAInCluster(
		ctx,
		kubernetesClient,
		*reqData.ClusterID,
		clusterData.LatestHPAAPIVersion,
	)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	eventData := &UCEntity.Event{
		Name:              *reqData.Name,
		ExecuteConfigAt:   *reqData.ExecuteConfigAt,
		WatchingAt:        *reqData.WatchingAt,
		StartTime:         *reqData.StartTime,
		EndTime:           *reqData.EndTime,
		CalculateNodePool: *reqData.CalculateNodePool,
	}
	eventData.Cluster.ID = *reqData.ClusterID

	eventID, err := e.eventUC.RegisterEvents(tx, eventData)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	var HPAConfigs []UCEntity.EventModifiedHPAConfigData
	for _, hpaConfig := range reqData.ModifiedHPAConfigs {
		found := false
		for _, HPA := range HPAs {
			if *hpaConfig.Name == HPA.Name && *hpaConfig.Namespace == HPA.Namespace {
				found = true
				break
			}
		}
		if found {
			HPAConfigs = append(
				HPAConfigs, UCEntity.EventModifiedHPAConfigData{
					Name:        *hpaConfig.Name,
					Namespace:   *hpaConfig.Namespace,
					MinReplicas: hpaConfig.MinReplicas,
					MaxReplicas: *hpaConfig.MaxReplicas,
				},
			)
		}
	}

	_, err = e.scheduledHPAConfigUC.RegisterModifiedHPAConfigs(tx, HPAConfigs, eventID)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	tx.Commit()

	return e.successResponse(c, response.EventCreationResponse{EventID: eventID})

}

func (e *event) ListEventByCluster(c *fiber.Ctx) error {
	reqData := &request.EventListRequest{}
	err := c.QueryParser(reqData)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	err = e.validatorInst.Struct(reqData)
	if err != nil {
		return e.errorResponse(c, errorConstant.InvalidQueryParam)
	}

	ctx := c.Context()
	tx := e.db.WithContext(ctx)

	events, err := e.eventUC.ListEventByClusterID(tx, *reqData.ClusterID)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	responseData := make([]response.EventSimpleResponse, 0)
	for _, event := range events {
		responseData = append(
			responseData, response.EventSimpleResponse{
				ID:        event.ID,
				Name:      event.Name,
				StartTime: event.StartTime,
				EndTime:   event.EndTime,
				Status:    event.Status,
			},
		)
	}

	return e.successResponse(c, responseData)

}

func (e *event) UpdateEvent(c *fiber.Ctx) error {
	req := &request.UpdateEventDataRequest{}
	if err := c.BodyParser(req); err != nil {
		return e.errorResponse(c, err.Error())
	}

	if err := e.validatorInst.Struct(req); err != nil {
		return e.errorResponse(c, errorConstant.InvalidRequestBody)
	}

	ctx := c.Context()
	db := e.db.WithContext(ctx)
	tx := db.Begin()

	eventData, err := e.eventUC.GetEventByID(db, *req.EventID)
	if err != nil {
		return e.errorResponse(c, errorConstant.EventNotExist)
	}

	if eventData.Name != *req.Name {
		eventData.Name = *req.Name
	}

	if req.CalculateNodePool != nil {
		eventData.CalculateNodePool = *req.CalculateNodePool
	}

	eventData.StartTime = *req.StartTime
	eventData.EndTime = *req.EndTime
	eventData.ExecuteConfigAt = *req.ExecuteConfigAt
	eventData.WatchingAt = *req.WatchingAt

	if err := e.eventUC.UpdateEvent(tx, eventData); err != nil {
		return e.errorResponse(c, err.Error())
	}

	if err := e.scheduledHPAConfigUC.DeleteEventModifiedHPAConfigs(tx, eventData.ID); err != nil {
		return e.errorResponse(c, err.Error())
	}

	var newModifiedHPAConfigs []UCEntity.EventModifiedHPAConfigData
	for _, hpaConfig := range req.ModifiedHPAConfigs {
		newModifiedHPAConfigs = append(
			newModifiedHPAConfigs, UCEntity.EventModifiedHPAConfigData{
				Name:        *hpaConfig.Name,
				Namespace:   *hpaConfig.Namespace,
				MinReplicas: hpaConfig.MinReplicas,
				MaxReplicas: *hpaConfig.MaxReplicas,
			},
		)
	}

	_, err = e.scheduledHPAConfigUC.RegisterModifiedHPAConfigs(
		tx,
		newModifiedHPAConfigs,
		eventData.ID,
	)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	tx.Commit()

	res := &response.EventCreationResponse{EventID: eventData.ID}
	return e.successResponse(c, res)
}

func (e *event) GetDetailedEvent(c *fiber.Ctx) error {
	eventIDStr := c.Params("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return e.errorResponse(c, fmt.Sprintf(errorConstant.ParamInvalid, "event_id"))
	}

	ctx := c.Context()
	db := e.db.WithContext(ctx)

	eventData, err := e.eventUC.GetDetailedEventData(db, eventID)
	if err != nil {
		return e.errorResponse(c, errorConstant.EventNotExist)
	}

	var modifiedHPAConfigRes []response.ModifiedHPAConfig
	for _, hpa := range eventData.EventModifiedHPAConfigData {
		modifiedHPAConfigRes = append(
			modifiedHPAConfigRes, response.ModifiedHPAConfig{
				ID:          hpa.ID,
				Name:        hpa.Name,
				Namespace:   hpa.Namespace,
				MinReplicas: hpa.MinReplicas,
				MaxReplicas: hpa.MaxReplicas,
			},
		)
	}

	updatedNodePools, err := e.statisticUC.GetAllUpdatedNodePoolByEvent(db, eventID)
	if err != nil {
		return e.errorResponse(c, errorConstant.EventNotExist)
	}

	updatedNodePoolRes := make([]response.UpdatedNodePool, 0)
	for _, updatedNodePool := range updatedNodePools {
		updatedNodePoolRes = append(
			updatedNodePoolRes, response.UpdatedNodePool{
				ID:           updatedNodePool.ID,
				NodePoolName: updatedNodePool.NodePoolName,
				MaxNode:      updatedNodePool.MaxNode,
			},
		)
	}

	res := &response.EventDetailedResponse{
		EventSimpleResponse: response.EventSimpleResponse{
			ID:        eventData.ID,
			Name:      eventData.Name,
			StartTime: eventData.StartTime,
			EndTime:   eventData.EndTime,
			Status:    eventData.Status,
		},
		CreatedAt: eventData.CreatedAt,
		UpdatedAt: eventData.UpdatedAt,
		Cluster: response.Cluster{
			ID:             &eventData.Cluster.ID,
			Name:           eventData.Cluster.Name,
			Datacenter:     eventData.Cluster.Datacenter.Datacenter,
			DatacenterName: eventData.Cluster.Datacenter.Name,
		},
		ModifiedHPAConfigs: modifiedHPAConfigRes,
		UpdatedNodePools:   updatedNodePoolRes,
		CalculateNodePool:  eventData.CalculateNodePool,
		ExecuteConfigAt:    eventData.ExecuteConfigAt,
		WatchingAt:         eventData.WatchingAt,
	}

	return e.successResponse(c, res)
}

func (e *event) DeleteEvent(c *fiber.Ctx) error {
	eventIDStr := c.Params("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return e.errorResponse(c, fmt.Sprintf(errorConstant.ParamInvalid, "event_id"))
	}

	ctx := c.Context()
	db := e.db.WithContext(ctx)
	tx := db.Begin()

	_, err = e.eventUC.GetEventByID(db, eventID)
	if err != nil {
		return e.errorResponse(c, errorConstant.EventNotExist)
	}

	err = e.eventUC.DeleteEvent(tx, eventID)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	err = e.scheduledHPAConfigUC.SoftDeleteEventModifiedHPAConfigs(tx, eventID)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	tx.Commit()

	return e.successResponse(c, constant.ActionDone)
}

func (e *event) ListNodePoolStatusByUpdatedNodePool(c *fiber.Ctx) error {
	updatedNodePoolIDStr := c.Params("updated_node_pool_id")
	updatedNodePoolID, err := uuid.Parse(updatedNodePoolIDStr)
	if err != nil {
		return e.errorResponse(c, fmt.Sprintf(errorConstant.ParamInvalid, "updated_node_pool_id"))
	}

	ctx := c.Context()
	db := e.db.WithContext(ctx)

	nodePoolStatuses, err := e.statisticUC.GetAllNodePoolStatusByUpdatedNodePoolID(
		db,
		updatedNodePoolID,
	)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	var resp []response.NodePoolStatus
	for _, nodePoolStatus := range nodePoolStatuses {
		resp = append(
			resp, response.NodePoolStatus{
				CreatedAt: nodePoolStatus.CreatedAt,
				Count:     nodePoolStatus.Count,
			},
		)
	}

	return e.successResponse(c, resp)
}

func (e *event) ListHPAStatusByScheduledHPAConfig(c *fiber.Ctx) error {
	scheduledHPAConfigIDStr := c.Params("scheduled_hpa_config_id")
	scheduledHPAConfigID, err := uuid.Parse(scheduledHPAConfigIDStr)
	if err != nil {
		return e.errorResponse(
			c,
			fmt.Sprintf(errorConstant.ParamInvalid, "scheduled_hpa_config_id"),
		)
	}

	ctx := c.Context()
	db := e.db.WithContext(ctx)

	hpaStatuses, err := e.statisticUC.GetAllHPAStatusByScheduledHPAConfigID(
		db,
		scheduledHPAConfigID,
	)
	if err != nil {
		return e.errorResponse(c, err.Error())
	}

	var resp []response.HPAStatus
	for _, nodePoolStatus := range hpaStatuses {
		resp = append(
			resp, response.HPAStatus{
				CreatedAt:           nodePoolStatus.CreatedAt,
				Replicas:            nodePoolStatus.Replicas,
				ReadyReplicas:       nodePoolStatus.ReadyReplicas,
				UnavailableReplicas: nodePoolStatus.UnavailableReplicas,
				AvailableReplicas:   nodePoolStatus.AvailableReplicas,
			},
		)
	}

	return e.successResponse(c, resp)
}
