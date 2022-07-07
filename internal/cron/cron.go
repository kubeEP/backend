package cron

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	v1Apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/api/autoscaling/v2beta2"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Cron interface {
	Start()
}

type cron struct {
	eventUC              useCase.Event
	clusterUC            useCase.Cluster
	gcpClusterUC         useCase.GCPCluster
	gcpDatacenterUC      useCase.GCPDatacenter
	scheduledHPAConfigUC useCase.ScheduledHPAConfig
	updatedNodePoolUC    useCase.Statistic
	tx                   *gorm.DB
}

func newCron(
	eventUC useCase.Event,
	clusterUC useCase.Cluster,
	gcpClusterUC useCase.GCPCluster,
	gcpDatacenterUC useCase.GCPDatacenter,
	scheduledHPAConfigUC useCase.ScheduledHPAConfig,
	updatedNodePoolUC useCase.Statistic,
	tx *gorm.DB,
) Cron {
	return &cron{
		eventUC:              eventUC,
		tx:                   tx,
		clusterUC:            clusterUC,
		gcpClusterUC:         gcpClusterUC,
		gcpDatacenterUC:      gcpDatacenterUC,
		scheduledHPAConfigUC: scheduledHPAConfigUC,
		updatedNodePoolUC:    updatedNodePoolUC,
	}
}

func (c *cron) handleExecEventError(db *gorm.DB, e *UCEntity.Event, errMsg string) {
	e.Status = model.EventFailed
	e.Message = errMsg
	err := c.eventUC.UpdateEvent(db, e)
	if err != nil {
		log.Errorf("[EventCronJob] Error Update Event : %s", err.Error())
	}
	log.Errorf("[EventCronJob] Event : %s, Error : %s", e.Name, errMsg)
}

func (c *cron) handleWatchEvent(db *gorm.DB, e *UCEntity.Event, errMsg string) {
	e.Message = errMsg
	err := c.eventUC.UpdateEvent(db, e)
	if err != nil {
		log.Errorf("[EventCronJob] Error Update Event : %s", err.Error())
	}
	log.Errorf("[EventCronJob] Watching event : %s, Error : %s", e.Name, errMsg)
}

func (c *cron) watchNodePool(
	client kubernetes.Interface,
	db *gorm.DB,
	provider model.DatacenterProvider,
	event *UCEntity.Event,
	now time.Time,
	ctx context.Context,
	updatedNodePoolMap map[string]uuid.UUID,
) {
	nodes, err := client.CoreV1().Nodes().List(ctx, v1Option.ListOptions{})
	if err != nil {
		log.Errorf(
			"[EventCronJob] Watching event : %s, Watch node pool error : %s",
			event.Name,
			err.Error(),
		)
		return
	}
	nodeCounts := map[string]int32{}
	for _, node := range nodes.Items {
		nodeLabels := node.Labels
		var nodePoolName string
		switch provider {
		case model.GCP:
			nodePoolName = nodeLabels[constant.GCPNodePoolLabel]
			nodeCounts[nodePoolName] += 1
		}
	}

	var nodePoolStatusObjects []model.NodePoolStatus
	for nodePoolName, nodeCount := range nodeCounts {
		nodePoolStatus := model.NodePoolStatus{
			CreatedAt: now,
			NodeCount: nodeCount,
		}
		nodePoolStatus.UpdatedNodePoolID.SetUUID(updatedNodePoolMap[nodePoolName])
		nodePoolStatusObjects = append(nodePoolStatusObjects, nodePoolStatus)
	}

	err = db.Create(&nodePoolStatusObjects).Error
	if err != nil {
		log.Errorf(
			"[EventCronJob] Watching event : %s, Watch node pool error : %s",
			event.Name,
			err.Error(),
		)
	}
	log.Infof(
		"[EventCronJob] Watching event : %s, Watching node pool at : %s",
		event.Name,
		now,
	)
}

func (c *cron) watchHPA(
	deploymentDataMapFunc func(ctx context.Context) (map[string]*DeploymentPodData, error),
	db *gorm.DB,
	event *UCEntity.Event,
	scheduledHPAConfigs []*UCEntity.EventModifiedHPAConfigData,
	now time.Time,
	ctx context.Context,
) {
	deploymentDataMap, err := deploymentDataMapFunc(ctx)
	if err != nil {
		log.Errorf(
			"[EventCronJob] Watching event : %s, Watch hpa error : %s",
			event.Name,
			err.Error(),
		)
		return
	}

	var selectedHPAStatuses []model.HPAStatus
	for _, scheduledHPAConfig := range scheduledHPAConfigs {
		key := fmt.Sprintf(
			constant.NameAndNamespaceKeyFormat,
			scheduledHPAConfig.Name,
			scheduledHPAConfig.Namespace,
		)
		data := deploymentDataMap[key]
		hpaStatus := model.HPAStatus{
			CreatedAt:           now,
			Replicas:            data.Replicas,
			AvailableReplicas:   data.AvailableReplicas,
			UnavailableReplicas: data.UnavailableReplicas,
			ReadyReplicas:       data.ReadyReplicas,
		}
		hpaStatus.ScheduledHPAConfigID.SetUUID(scheduledHPAConfig.ID)
		selectedHPAStatuses = append(selectedHPAStatuses, hpaStatus)
	}

	err = db.Create(&selectedHPAStatuses).Error
	if err != nil {
		log.Errorf(
			"[EventCronJob] Watching event : %s, Watch hpa error : %s",
			event.Name,
			err.Error(),
		)
		return
	}

	log.Infof(
		"[EventCronJob] Watching event : %s, Watching hpa at : %s",
		event.Name,
		now,
	)
}

func (c *cron) watchEvent(e *UCEntity.Event, db *gorm.DB, ctx context.Context) {
	log.Infof("[EventCronJob] Watching event %s", e.Name)
	e.Status = model.EventWatching

	err := c.eventUC.UpdateEvent(db, e)
	if err != nil {
		log.Errorf("[EventCronJob] Error update event : %s", err.Error())
		return
	}

	clusterID := e.Cluster.ID
	clusterData, err := c.clusterUC.GetClusterAndDatacenterDataByClusterID(db, clusterID)
	if err != nil {
		c.handleWatchEvent(db, e, err.Error())
		return
	}
	datacenter := clusterData.Datacenter.Datacenter
	var kubernetesClient kubernetes.Interface

	// Get Clients
	switch datacenter {
	case model.GCP:
		kubernetesClient, _, err = c.getAllGCPClient(ctx, clusterData)
		if err != nil {
			c.handleWatchEvent(db, e, err.Error())
			return
		}
	}

	scheduledHPAConfigs, err := c.scheduledHPAConfigUC.ListScheduledHPAConfigByEventID(db, e.ID)
	if err != nil {
		c.handleWatchEvent(db, e, err.Error())
		return
	}

	allHPAK8sObject, err := c.clusterUC.GetAllK8sHPAObjectInCluster(
		ctx,
		kubernetesClient,
		clusterID,
		clusterData.LatestHPAAPIVersion,
	)
	if err != nil {
		c.handleWatchEvent(db, e, err.Error())
		return
	}

	mapHPAScaleTargetRef := map[string]interface{}{}
	for _, data := range allHPAK8sObject {
		for _, scheduledHPAConfig := range scheduledHPAConfigs {
			switch h := data.HPAObject.(type) {
			case v1.HorizontalPodAutoscaler:
				if scheduledHPAConfig.Name == h.Name && scheduledHPAConfig.Namespace == h.Namespace {
					mapHPAScaleTargetRef[fmt.Sprintf(
						constant.NameAndNamespaceKeyFormat,
						h.Name,
						h.Namespace,
					)] = h.Spec.ScaleTargetRef
					break
				}
			case v2beta1.HorizontalPodAutoscaler:
				if scheduledHPAConfig.Name == h.Name && scheduledHPAConfig.Namespace == h.Namespace {
					mapHPAScaleTargetRef[fmt.Sprintf(
						constant.NameAndNamespaceKeyFormat,
						h.Name,
						h.Namespace,
					)] = h.Spec.ScaleTargetRef
					break
				}
			case v2beta2.HorizontalPodAutoscaler:
				if scheduledHPAConfig.Name == h.Name && scheduledHPAConfig.Namespace == h.Namespace {
					mapHPAScaleTargetRef[fmt.Sprintf(
						constant.NameAndNamespaceKeyFormat,
						h.Name,
						h.Namespace,
					)] = h.Spec.ScaleTargetRef
					break
				}
			}
		}
	}

	getAllDeploymentsFunc := func(ctx context.Context) (map[string]*DeploymentPodData, error) {
		mapDeploymentsPodData := map[string]*DeploymentPodData{}
		errGroup, ctxEg := errgroup.WithContext(ctx)
		for key, val := range mapHPAScaleTargetRef {
			nameSplit := strings.Split(key, "|")
			data := &DeploymentPodData{
				Name:      nameSplit[0],
				Namespace: nameSplit[1],
			}
			mapDeploymentsPodData[key] = data
			loadFunc := func(
				namespace string,
				scaleTargetRef interface{},
				data *DeploymentPodData,
			) func() error {
				return func() error {
					res, err := c.clusterUC.ResolveScaleTargetRef(
						ctxEg,
						kubernetesClient,
						scaleTargetRef,
						namespace,
					)
					if err != nil {
						if ctxEg.Err() != nil {
							return nil
						}

						return err
					}

					switch resolveRes := res.(type) {
					case *v1Apps.Deployment:
						status := resolveRes.Status
						data.Replicas = status.Replicas
						data.UnavailableReplicas = status.UnavailableReplicas
						data.ReadyReplicas = status.ReadyReplicas
						data.AvailableReplicas = status.AvailableReplicas
					}

					return nil
				}
			}

			errGroup.Go(
				loadFunc(
					nameSplit[1],
					val,
					data,
				),
			)
		}

		if err := errGroup.Wait(); err != nil {
			return nil, err
		}

		return mapDeploymentsPodData, nil
	}

	updatedNodePools, err := c.updatedNodePoolUC.GetAllUpdatedNodePoolByEvent(db, e.ID)
	if err != nil {
		c.handleWatchEvent(db, e, err.Error())
		return
	}
	updatedNodePoolMap := map[string]uuid.UUID{}
	for _, updatedNodePool := range updatedNodePools {
		updatedNodePoolMap[updatedNodePool.NodePoolName] = updatedNodePool.ID
	}

	endTime := e.EndTime
	watcherTicker := time.NewTicker(30 * time.Second)
	defer watcherTicker.Stop()
	for {
		select {
		case now := <-watcherTicker.C:
			if now.After(endTime) {
				return
			}

			go c.watchNodePool(kubernetesClient, db, datacenter, e, now, ctx, updatedNodePoolMap)
			go c.watchHPA(getAllDeploymentsFunc, db, e, scheduledHPAConfigs, now, ctx)
		case <-ctx.Done():
			return
		}
	}

}

func (c *cron) Start() {
	log.Infof("Starting event cron job")
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	db := c.tx.WithContext(ctx)
	mainTicker := time.NewTicker(1 * time.Minute)
	defer mainTicker.Stop()
	for {
		select {
		case now := <-mainTicker.C:
			go func() {
				pendingEvents, err := c.eventUC.GetAllPendingExecutableEvent(db, now)
				if err != nil {
					log.Errorf(
						"[EventCronJob] Error getting pending executable events : %s",
						err.Error(),
					)
				}
				if len(pendingEvents) != 0 && err == nil {
					for _, pendingEvent := range pendingEvents {
						switch pendingEvent.Cluster.Datacenter.Datacenter {
						case model.GCP:
							go c.execGCPEvent(pendingEvent, db, ctx)

						}
					}
				}
			}()

			go func() {
				prescaledEvents, err := c.eventUC.GetAllPrescaledEvent(db, now)
				if err != nil {
					log.Errorf(
						"[EventCronJob] Error getting prescaled events : %s",
						err.Error(),
					)
				}
				if len(prescaledEvents) != 0 && err == nil {
					for _, prescaledEvent := range prescaledEvents {
						go c.watchEvent(prescaledEvent, db, ctx)
					}
				}
			}()

			go func() {
				err := c.eventUC.FinishAllWatchedEvent(db, now)
				if err != nil {
					log.Errorf("[EventCronJob] Error update watched events : %s", err.Error())
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}
