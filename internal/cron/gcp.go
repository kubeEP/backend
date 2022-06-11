package cron

import (
	"context"
	"errors"
	"fmt"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/util"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/googleapis/container/v1"
	"gorm.io/gorm"
	v1Apps "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v1"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/api/autoscaling/v2beta2"
	v1Core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"math"
	"strings"
	"sync"
	"time"
)

func (c *cron) getAllGCPClient(
	ctx context.Context,
	clusterData *UCEntity.ClusterData,
) (kubernetes.Interface, *GCPClients, error) {
	datacenter := clusterData.Datacenter.Datacenter
	if datacenter != model.GCP {
		return nil, nil, errors.New(errorConstant.DatacenterMismatch)
	}
	datacenterName := clusterData.Datacenter.Name
	datacenterData := UCEntity.DatacenterData{
		Credentials: clusterData.Datacenter.Credentials,
		Name:        datacenterName,
	}
	googleCredential, err := c.gcpDatacenterUC.GetGoogleCredentials(
		ctx,
		datacenterData,
	)
	gcpClusterClient, err := c.gcpClusterUC.GetGoogleClusterClient(ctx, googleCredential)
	if err != nil {
		return nil, nil, err
	}
	gcpIgmClient, err := c.gcpClusterUC.GetGoogleInstanceGroupManagersClient(ctx, googleCredential)
	if err != nil {
		return nil, nil, err
	}
	gcpInstanceTemplatesClient, err := c.gcpClusterUC.GetGoogleInstanceTemplatesClient(
		ctx,
		googleCredential,
	)
	if err != nil {
		return nil, nil, err
	}
	c.gcpClusterUC.RegisterGoogleCredentials(datacenterName, googleCredential)
	kubernetesClient, err := c.gcpClusterUC.GetKubernetesClusterClient(
		datacenterName,
		clusterData,
	)
	if err != nil {
		return nil, nil, err
	}
	return kubernetesClient, &GCPClients{
		clusterClient:               gcpClusterClient,
		instanceGroupManagersClient: gcpIgmClient,
		instanceTemplatesClient:     gcpInstanceTemplatesClient,
	}, nil
}

func (c *cron) execGCPEvent(e *UCEntity.Event, db *gorm.DB, ctx context.Context) {
	log.Infof("[EventCronJob] Executing event %s", e.Name)
	e.Status = model.EventExecuting

	err := c.eventUC.UpdateEvent(db, e)
	if err != nil {
		log.Errorf("[EventCronJob] Error Update Event : %s", err.Error())
		return
	}

	if !e.CalculateNodePool {
		log.Infof("[EventCronJob] Event %s, skipping node pool calculation", e.Name)
	}

	clusterID := e.Cluster.ID
	clusterData, err := c.clusterUC.GetClusterAndDatacenterDataByClusterID(db, clusterID)
	if err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	// Get Clients
	kubernetesClient, googleClients, err := c.getAllGCPClient(ctx, clusterData)
	if err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	googleContainerClient := googleClients.clusterClient

	modifiedHPAs, err := c.scheduledHPAConfigUC.ListScheduledHPAConfigByEventID(db, e.ID)
	if err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	// Check HPA
	log.Infof("[EventCronJob] Event : %s, Checking HPAs", e.Name)
	existingK8sHPA, err := c.clusterUC.GetAllK8sHPAObjectInCluster(
		ctx,
		kubernetesClient,
		clusterID,
		clusterData.LatestHPAAPIVersion,
	)
	if err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	// Search selected and unselected hpa
	var selectedK8sHPAs []interface{}
	var unselectedK8sHPAs []interface{}
	var selectedK8sHPANames []string
	var unselectedK8sHPANames []string
	var existingModifiedHPAs []*UCEntity.EventModifiedHPAConfigData
	modifiedHPAMap := map[string]*UCEntity.EventModifiedHPAConfigData{}
	for _, modifiedHPA := range modifiedHPAs {
		key := fmt.Sprintf(constant.NameNSKeyFormat, modifiedHPA.Name, modifiedHPA.Namespace)
		modifiedHPAMap[key] = modifiedHPA
	}

	for _, data := range existingK8sHPA {
		var name, namespace string
		var deepCopy interface{}
		switch h := data.HPAObject.(type) {
		case v1.HorizontalPodAutoscaler:
			name = h.Name
			namespace = h.Namespace
			deepCopy = h.DeepCopy()
		case v2beta1.HorizontalPodAutoscaler:
			name = h.Name
			namespace = h.Namespace
			deepCopy = h.DeepCopy()
		case v2beta2.HorizontalPodAutoscaler:
			name = h.Name
			namespace = h.Namespace
			deepCopy = h.DeepCopy()
		default:
			continue
		}
		key := fmt.Sprintf(constant.NameNSKeyFormat, name, namespace)
		modifiedHPA, ok := modifiedHPAMap[key]
		if ok && modifiedHPA != nil {
			selectedK8sHPAs = append(selectedK8sHPAs, deepCopy)
			selectedK8sHPANames = append(selectedK8sHPANames, key)
			existingModifiedHPAs = append(existingModifiedHPAs, modifiedHPA)
			delete(modifiedHPAMap, key)
			continue
		}
		unselectedK8sHPAs = append(unselectedK8sHPAs, deepCopy)
		unselectedK8sHPANames = append(unselectedK8sHPANames, key)
	}

	//Give error message to missing hpa
	for _, modifiedHPA := range modifiedHPAMap {
		err := c.scheduledHPAConfigUC.UpdateScheduledHPAConfigStatusMessage(
			db,
			modifiedHPA.ID,
			model.HPAUpdateFailed,
			"hpa not found",
		)
		if err != nil {
			log.Errorf(
				"[EventCronJob] Event : %s, Error Update HPA %s Namespace %s : %s",
				e.Name,
				modifiedHPA.Name,
				modifiedHPA.Namespace,
				err.Error(),
			)
		}
	}

	if len(selectedK8sHPAs) == 0 {
		c.handleExecEventError(db, e, "no hpa exist")
		return
	}

	log.Infof(
		"[EventCronJob] Event : %s, Selected HPAs:\n%s\nUnselected HPAs:\n%s",
		e.Name,
		strings.Join(selectedK8sHPANames, "\n"),
		strings.Join(unselectedK8sHPANames, "\n"),
	)

	// Parse GCP Cluster Name
	clusterMetadata := strings.Split(clusterData.Name, "_")
	project := clusterMetadata[1]
	location := clusterMetadata[3]
	name := clusterMetadata[2]

	// Get GCP Node Pools
	googleClusterData, err := c.gcpClusterUC.GetGCPClusterObject(
		ctx,
		googleContainerClient,
		project,
		location,
		name,
	)
	if err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	// Get Linux Daemonsets and Calculate Required Resources
	var daemonSetsDataList []*DaemonSetData
	if e.CalculateNodePool {
		log.Infof("[EventCronJob] Event : %s, Calculate daemonsets resources", e.Name)
		daemonSetsData, err := c.clusterUC.GetAllDaemonSetsInNamespace(
			ctx,
			kubernetesClient,
			"",
		)
		if err != nil {
			c.handleExecEventError(db, e, err.Error())
			return
		}
		daemonSets := daemonSetsData.DaemonSetListObject

		for _, daemonSet := range daemonSets.Items {
			spec := daemonSet.Spec.Template.Spec
			totalRequestedMemory := float64(0)
			totalRequestedCPU := float64(0)
			for _, containerData := range spec.Containers {
				totalRequestedMemory += containerData.Resources.Requests.Memory().AsApproximateFloat64()
				totalRequestedCPU += containerData.Resources.Requests.Cpu().AsApproximateFloat64()
			}
			var nodeAffinity *v1Core.NodeAffinity
			if spec.Affinity != nil {
				if spec.Affinity.NodeAffinity != nil {
					nodeAffinity = spec.Affinity.NodeAffinity
				}
			}
			daemonSetsDataList = append(
				daemonSetsDataList, &DaemonSetData{
					NodeSelector:    labels.Set(spec.NodeSelector).AsSelector(),
					NodeAffinity:    nodeAffinity,
					RequestedMemory: totalRequestedMemory,
					RequestedCPU:    totalRequestedCPU,
					Name:            daemonSet.Name,
					Namespace:       daemonSet.Namespace,
				},
			)
			log.Infof(
				"[EventCronJob] Event : %s, Registering daemonset %s namespace %s, %f requested memory, %f requested CPU",
				e.Name,
				daemonSet.Name,
				daemonSet.Namespace,
				totalRequestedMemory,
				totalRequestedCPU,
			)
		}
	}

	// Get Maximum Resources each Node Pools
	log.Infof(
		"[EventCronJob] Event : %s, Calculate maximum available resources in node pools",
		e.Name,
	)
	nodePools := googleClusterData.ClusterObject.NodePools
	nodePoolsMaxResources := map[string]*NodePoolResourceData{}
	nodePoolsRequestedResources := map[string]*NodePoolRequestedResourceData{}
	nodePoolsMap := map[string]*container.NodePool{}
	var updatedNodePools []*model.UpdatedNodePool
	var nodePoolsList []string
	errGroup, ctxEg := errgroup.WithContext(ctx)
	for _, nodePool := range nodePools {
		nodePoolsList = append(nodePoolsList, nodePool.Name)
		nodePoolsRequestedResources[nodePool.Name] = &NodePoolRequestedResourceData{}
		resourceData := &NodePoolResourceData{}
		nodePoolsMaxResources[nodePool.Name] = resourceData
		nodePoolsMap[nodePool.Name] = nodePool

		updatedNodePool := &model.UpdatedNodePool{
			NodePoolName: nodePool.Name,
		}
		if nodePool.Autoscaling != nil {
			updatedNodePool.MaxNode = nodePool.Autoscaling.MaxNodeCount
		}
		updatedNodePool.EventID.SetUUID(e.ID)

		updatedNodePools = append(updatedNodePools, updatedNodePool)

		if e.CalculateNodePool {
			loadFunc := func(nP *container.NodePool, rD *NodePoolResourceData) func() error {
				return func() error {
					nodePoolMaxPods := nP.MaxPodsConstraint.MaxPodsPerNode
					nodePoolMaxNode := nP.Autoscaling.MaxNodeCount

					// Fetch nodepool labels from existing node
					nodeData, err := c.gcpClusterUC.GetNodesFromGCPNodePool(
						ctxEg,
						kubernetesClient,
						nP.Name,
					)
					if err != nil {
						if ctxEg.Err() != nil {
							return nil
						}
						log.Errorf(
							"[EventCronJob] Event : %s, Node pool %s, Error : %s",
							e.Name,
							nP.Name,
							err.Error(),
						)
						return err
					}
					nodes := nodeData.NodeListObject
					node := nodes.Items[0]
					availablePods := nodePoolMaxPods
					rD.NodeLabels = node.Labels
					totalMatchesDaemonSet := int64(0)
					totalDaemonSetsRequestedCPU := float64(0)
					totalDaemonSetsRequestedMemory := float64(0)
					var matchesDaemonSet []string
					for _, daemonSet := range daemonSetsDataList {
						nodePoolMatch, err := util.CheckPodNodePoolMatch(
							rD.NodeLabels,
							daemonSet.NodeAffinity,
							daemonSet.NodeSelector,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							log.Errorf(
								"[EventCronJob] Event : %s, Node pool %s, Error : %s",
								e.Name,
								nP.Name,
								err.Error(),
							)
							return err
						}
						if nodePoolMatch {
							totalDaemonSetsRequestedCPU += daemonSet.RequestedCPU
							totalDaemonSetsRequestedMemory += daemonSet.RequestedMemory
							totalMatchesDaemonSet += 1
							matchesDaemonSet = append(
								matchesDaemonSet,
								fmt.Sprintf(
									constant.NameNSKeyFormat,
									daemonSet.Name,
									daemonSet.Namespace,
								),
							)
						}
					}

					log.Infof(
						"[EventCronJob] Event : %s, Node pool %s, %d matches daemonset with %f requested memory and %f requested cpu\nDaemonset list :\n%s",
						e.Name,
						nP.Name,
						totalMatchesDaemonSet,
						totalDaemonSetsRequestedMemory,
						totalDaemonSetsRequestedCPU,
						strings.Join(matchesDaemonSet, "\n"),
					)

					rD.AvailablePods = availablePods - totalMatchesDaemonSet
					rD.MaxAvailablePods = availablePods * int64(nodePoolMaxNode)
					allocatableCPU := node.Status.Allocatable.Cpu().AsApproximateFloat64()
					allocatableMemory := node.Status.Allocatable.Memory().AsApproximateFloat64()
					availableCPU := allocatableCPU - totalDaemonSetsRequestedCPU
					availableMemory := allocatableMemory - totalDaemonSetsRequestedMemory
					rD.AvailableCPU = availableCPU
					rD.AvailableMemory = availableMemory
					rD.MaxAvailableCPU = availableCPU * float64(nodePoolMaxNode)
					rD.MaxAvailableMemory = availableMemory * float64(nodePoolMaxNode)
					rD.CurrentNodeCount = len(nodes.Items)

					log.Infof(
						"[EventCronJob] Event : %s, Node pool %s has maximum %d available pods, maximum %f available cpu and maximum %f available memory",
						e.Name,
						nP.Name,
						rD.MaxAvailablePods,
						rD.MaxAvailableCPU,
						rD.MaxAvailableMemory,
					)

					return nil
				}
			}
			errGroup.Go(loadFunc(nodePool, resourceData))
		}
	}

	if err := errGroup.Wait(); err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	// Calculate Required Resource
	var nodePoolRequestedResourceLock sync.Mutex
	var deploymentsMap map[string]v1Apps.Deployment
	errGroup, ctxEg = errgroup.WithContext(ctx)
	if e.CalculateNodePool {
		log.Infof("[EventCronJob] Event : %s, Calculate required resources", e.Name)

		deploymentsMap = map[string]v1Apps.Deployment{}
		var deploymentNames []string

		log.Infof("[EventCronJob] Event : %s, Fetching deployments", e.Name)
		deploymentsData, err := c.clusterUC.GetAllDeployments(ctxEg, kubernetesClient, "")
		if err != nil {
			c.handleExecEventError(db, e, err.Error())
			return
		}

		deployments := deploymentsData.DeploymentListObject
		for _, deployment := range deployments.Items {
			key := fmt.Sprintf(constant.NameNSKeyFormat, deployment.Name, deployment.Namespace)
			deploymentNames = append(deploymentNames, key)
			deploymentsMap[key] = deployment
		}
		log.Infof(
			"[EventCronJob] Event : %s, Found deployments:\n%s",
			e.Name,
			strings.Join(deploymentNames, "\n"),
		)
	}

	log.Infof("[EventCronJob] Event : %s, Calculate selected HPA", e.Name)
	// Calculate Selected HPA
	for idx, selectedHPA := range selectedK8sHPAs {
		errGroup.Go(
			func(i int, hpa interface{}) func() error {
				return func() error {
					requestedModification := existingModifiedHPAs[i]
					var scaleTargetRef interface{}
					name := requestedModification.Name
					namespace := requestedModification.Namespace
					maxReplicas := requestedModification.MaxReplicas

					// Modify HPA, Get Target Ref and Namespace
					switch h := hpa.(type) {
					case *v1.HorizontalPodAutoscaler:
						h.Spec.MinReplicas = requestedModification.MinReplicas
						h.Spec.MaxReplicas = maxReplicas
						scaleTargetRef = h.Spec.ScaleTargetRef
					case *v2beta1.HorizontalPodAutoscaler:
						h.Spec.MinReplicas = requestedModification.MinReplicas
						h.Spec.MaxReplicas = maxReplicas
						scaleTargetRef = h.Spec.ScaleTargetRef
					case *v2beta2.HorizontalPodAutoscaler:
						h.Spec.MinReplicas = requestedModification.MinReplicas
						h.Spec.MaxReplicas = maxReplicas
						scaleTargetRef = h.Spec.ScaleTargetRef
					default:
						return errors.New(errorConstant.HPAVersionUnknown)
					}

					if e.CalculateNodePool {
						// Resolve Target Ref to Get Pods
						resolveRes, err := c.clusterUC.ResolveScaleTargetRefByDeploymentsMap(
							scaleTargetRef,
							namespace,
							deploymentsMap,
							true,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							log.Errorf(
								"[EventCronJob] Event : %s, Selected HPA %s Namespace %s, Error : %s",
								e.Name,
								name,
								namespace,
								err.Error(),
							)
							return err
						}

						// Get Labels Selector and Calculate Requested Resource
						var maxRequestedCPU, maxRequestedMemory float64
						nodeSelector := labels.Set(resolveRes.Spec.Template.Spec.NodeSelector).AsSelector()
						var nodeAffinity *v1Core.NodeAffinity
						if resolveRes.Spec.Template.Spec.Affinity != nil {
							if resolveRes.Spec.Template.Spec.Affinity.NodeAffinity != nil {
								nodeAffinity = resolveRes.Spec.Template.Spec.Affinity.NodeAffinity
							}
						}

						totalCpuRequested := float64(0)
						totalMemoryRequested := float64(0)
						containers := resolveRes.Spec.Template.Spec.Containers
						for _, containerSpec := range containers {
							totalCpuRequested += containerSpec.Resources.Requests.Cpu().AsApproximateFloat64()
							totalMemoryRequested += containerSpec.Resources.Requests.Memory().AsApproximateFloat64()
						}
						maxRequestedCPU = totalCpuRequested * float64(maxReplicas)
						maxRequestedMemory = totalMemoryRequested * float64(maxReplicas)

						//Resolve node selector and Find all node pools
						hpaNodePools := map[string]bool{}

						for nodePoolName, nodePoolResourceData := range nodePoolsMaxResources {
							nodeLabels := nodePoolResourceData.NodeLabels
							nodePoolMatch, err := util.CheckPodNodePoolMatch(
								nodeLabels,
								nodeAffinity,
								nodeSelector,
							)
							if err != nil {
								if ctxEg.Err() != nil {
									return nil
								}
								log.Errorf(
									"[EventCronJob] Event : %s, Selected HPA %s Namespace %s, Error : %s",
									e.Name,
									name,
									namespace,
									err.Error(),
								)
								return err
							}
							if nodePoolMatch {
								hpaNodePools[nodePoolName] = true
							}
						}

						// Calculate & Save requested resource data
						nodePoolRequestedResourceLock.Lock()
						defer nodePoolRequestedResourceLock.Unlock()

						var selectedNodePools []string

						for nodePoolName := range hpaNodePools {
							requestedResourceData := nodePoolsRequestedResources[nodePoolName]
							requestedResourceData.MaxPods += int64(maxReplicas)
							requestedResourceData.MaxCPU += maxRequestedCPU
							requestedResourceData.MaxMemory += maxRequestedMemory
							selectedNodePools = append(selectedNodePools, nodePoolName)
						}

						log.Infof(
							"[EventCronJob] Event : %s, Selected HPA %s namespace %s, maximum %d pods, maximum %f requested memory, maximum %f requested cpu\nNode pools:\n%s",
							e.Name,
							name,
							namespace,
							maxReplicas,
							maxRequestedMemory,
							maxRequestedCPU,
							strings.Join(selectedNodePools, "\n"),
						)
						return nil
					}

					log.Infof(
						"[EventCronJob] Event : %s, Selected HPA %s namespace %s, maximum %d pods",
						e.Name,
						name,
						namespace,
						maxReplicas,
					)

					return nil
				}
			}(idx, selectedHPA),
		)
	}

	if e.CalculateNodePool {
		// Calculate Unselected HPA
		for idx, unselectedHPA := range unselectedK8sHPAs {
			errGroup.Go(
				func(i int, hpa interface{}) func() error {
					return func() error {
						var scaleTargetRef interface{}
						var namespace, name string
						var maxReplicas int32

						// Modify HPA, Get Target Ref and Namespace
						switch h := hpa.(type) {
						case *v1.HorizontalPodAutoscaler:
							scaleTargetRef = h.Spec.ScaleTargetRef
							namespace = h.Namespace
							name = h.Name
							maxReplicas = h.Spec.MaxReplicas
						case *v2beta1.HorizontalPodAutoscaler:
							scaleTargetRef = h.Spec.ScaleTargetRef
							namespace = h.Namespace
							name = h.Name
							maxReplicas = h.Spec.MaxReplicas
						case *v2beta2.HorizontalPodAutoscaler:
							scaleTargetRef = h.Spec.ScaleTargetRef
							namespace = h.Namespace
							name = h.Name
							maxReplicas = h.Spec.MaxReplicas
						default:
							return errors.New(errorConstant.HPAVersionUnknown)
						}

						// Resolve Target Ref to Get Pods
						resolveRes, err := c.clusterUC.ResolveScaleTargetRefByDeploymentsMap(
							scaleTargetRef,
							namespace,
							deploymentsMap,
							true,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							log.Errorf(
								"[EventCronJob] Event : %s, Selected HPA %s Namespace %s, Error : %s",
								e.Name,
								name,
								namespace,
								err.Error(),
							)
							return err
						}

						// Get Labels Selector and Calculate Requested Resource
						var maxRequestedCPU, maxRequestedMemory float64
						nodeSelector := labels.Set(resolveRes.Spec.Template.Spec.NodeSelector).AsSelector()
						var nodeAffinity *v1Core.NodeAffinity
						if resolveRes.Spec.Template.Spec.Affinity != nil {
							if resolveRes.Spec.Template.Spec.Affinity.NodeAffinity != nil {
								nodeAffinity = resolveRes.Spec.Template.Spec.Affinity.NodeAffinity
							}
						}

						totalCpuRequested := float64(0)
						totalMemoryRequested := float64(0)
						containers := resolveRes.Spec.Template.Spec.Containers
						for _, containerSpec := range containers {
							totalCpuRequested += containerSpec.Resources.Requests.Cpu().AsApproximateFloat64()
							totalMemoryRequested += containerSpec.Resources.Requests.Memory().AsApproximateFloat64()
						}
						maxRequestedCPU = totalCpuRequested * float64(maxReplicas)
						maxRequestedMemory = totalMemoryRequested * float64(maxReplicas)

						//Resolve node selector and Find all node pools
						hpaNodePools := map[string]bool{}

						for nodePoolName, nodePoolResourceData := range nodePoolsMaxResources {
							nodeLabels := nodePoolResourceData.NodeLabels
							nodePoolMatch, err := util.CheckPodNodePoolMatch(
								nodeLabels,
								nodeAffinity,
								nodeSelector,
							)
							if err != nil {
								if ctxEg.Err() != nil {
									return nil
								}
								log.Errorf(
									"[EventCronJob] Event : %s, Unselected HPA %s Namespace %s, Error : %s",
									e.Name,
									name,
									namespace,
									err.Error(),
								)
								return err
							}
							if nodePoolMatch {
								hpaNodePools[nodePoolName] = true
							}
						}

						// Calculate & Save requested resource data
						nodePoolRequestedResourceLock.Lock()
						defer nodePoolRequestedResourceLock.Unlock()

						var selectedNodePools []string

						for nodePoolName := range hpaNodePools {
							requestedResourceData := nodePoolsRequestedResources[nodePoolName]
							requestedResourceData.MaxPods += int64(maxReplicas)
							requestedResourceData.MaxCPU += maxRequestedCPU
							requestedResourceData.MaxMemory += maxRequestedMemory
							selectedNodePools = append(selectedNodePools, nodePoolName)
						}

						log.Infof(
							"[EventCronJob] Event : %s, Unselected HPA %s namespace %s, maximum %d pods, maximum %f requested memory, maximum %f requested cpu\nNode pools:\n%s",
							e.Name,
							name,
							namespace,
							maxReplicas,
							maxRequestedMemory,
							maxRequestedCPU,
							strings.Join(selectedNodePools, "\n"),
						)

						return nil
					}
				}(idx, unselectedHPA),
			)
		}

		// Calculate remaining deployment
		for _, deployment := range deploymentsMap {
			errGroup.Go(
				func(d v1Apps.Deployment) func() error {
					return func() error {
						name := d.Name
						namespace := d.Namespace
						podCounts := d.Spec.Replicas
						if podCounts == nil {
							podCounts = &constant.MinimumPod
						}
						// Get Labels Selector and Calculate Requested Resource
						var maxRequestedCPU, maxRequestedMemory float64
						nodeSelector := labels.Set(d.Spec.Template.Spec.NodeSelector).AsSelector()
						var nodeAffinity *v1Core.NodeAffinity
						if d.Spec.Template.Spec.Affinity != nil {
							if d.Spec.Template.Spec.Affinity.NodeAffinity != nil {
								nodeAffinity = d.Spec.Template.Spec.Affinity.NodeAffinity
							}
						}

						totalCpuRequested := float64(0)
						totalMemoryRequested := float64(0)
						containers := d.Spec.Template.Spec.Containers
						for _, containerSpec := range containers {
							totalCpuRequested += containerSpec.Resources.Requests.Cpu().AsApproximateFloat64()
							totalMemoryRequested += containerSpec.Resources.Requests.Memory().AsApproximateFloat64()
						}
						maxRequestedCPU = totalCpuRequested * float64(*podCounts)
						maxRequestedMemory = totalMemoryRequested * float64(*podCounts)

						//Resolve node selector and Find all node pools
						deploymentNodePools := map[string]bool{}

						for nodePoolName, nodePoolResourceData := range nodePoolsMaxResources {
							nodeLabels := nodePoolResourceData.NodeLabels
							nodePoolMatch, err := util.CheckPodNodePoolMatch(
								nodeLabels,
								nodeAffinity,
								nodeSelector,
							)
							if err != nil {
								if ctxEg.Err() != nil {
									return nil
								}
								log.Errorf(
									"[EventCronJob] Event : %s, Deployment %s Namespace %s, Error : %s",
									e.Name,
									name,
									namespace,
									err.Error(),
								)
								return err
							}
							if nodePoolMatch {
								deploymentNodePools[nodePoolName] = true
							}
						}

						// Calculate & Save requested resource data
						nodePoolRequestedResourceLock.Lock()
						defer nodePoolRequestedResourceLock.Unlock()

						var selectedNodePools []string

						for nodePoolName := range deploymentNodePools {
							requestedResourceData := nodePoolsRequestedResources[nodePoolName]
							requestedResourceData.MaxPods += int64(*podCounts)
							requestedResourceData.MaxCPU += maxRequestedCPU
							requestedResourceData.MaxMemory += maxRequestedMemory
							selectedNodePools = append(selectedNodePools, nodePoolName)
						}

						log.Infof(
							"[EventCronJob] Event : %s, Deployment %s namespace %s, %d pods, maximum %f requested memory, maximum %f requested cpu\nNode pools:\n%s",
							e.Name,
							name,
							namespace,
							*podCounts,
							maxRequestedMemory,
							maxRequestedCPU,
							strings.Join(selectedNodePools, "\n"),
						)

						return nil
					}
				}(deployment),
			)
		}
	}

	if err := errGroup.Wait(); err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	if e.CalculateNodePool {
		// Calculate Requested Resource Each Node Pool and Update the Node Pool
		log.Infof(
			"[EventCronJob] Event : %s, Calculate needed pool based on requested resources",
			e.Name,
		)
		errGroup, ctxEg = errgroup.WithContext(ctx)
		var updateNodePoolLock sync.Mutex
		for idx, nodePoolName := range nodePoolsList {
			requestedResourceData := nodePoolsRequestedResources[nodePoolName]
			maxResourceData := nodePoolsMaxResources[nodePoolName]
			nodePool := nodePoolsMap[nodePoolName]
			errGroup.Go(
				func(
					reqResources *NodePoolRequestedResourceData,
					maxResources *NodePoolResourceData,
					nodePoolObj *container.NodePool,
					updatedNodePool *model.UpdatedNodePool,
				) func() error {
					return func() error {
						unfulfilledCPU := float64(0)
						if reqResources.MaxCPU > maxResources.MaxAvailableCPU {
							unfulfilledCPU = reqResources.MaxCPU - maxResources.MaxAvailableCPU
						}

						unfulfilledMemory := float64(0)
						if reqResources.MaxMemory > maxResources.MaxAvailableMemory {
							unfulfilledMemory = reqResources.MaxMemory - maxResources.MaxAvailableMemory
						}

						unfulfilledPods := int64(0)
						if reqResources.MaxPods > maxResources.MaxAvailablePods {
							unfulfilledPods = reqResources.MaxPods - maxResources.MaxAvailablePods
						}

						log.Infof(
							"[EventCronJob] Event : %s, Node pool %s, %f requested cpu (%f max available cpu), %f requested memory (%f max available memory), %d requested pods (%d max available pods)",
							e.Name,
							nodePoolObj.Name,
							reqResources.MaxCPU,
							maxResources.MaxAvailableCPU,
							reqResources.MaxMemory,
							maxResources.MaxAvailableMemory,
							reqResources.MaxPods,
							maxResources.MaxAvailablePods,
						)

						neededNodeBasedOnCPU := math.Ceil(unfulfilledCPU / maxResources.AvailableCPU)
						neededNodeBasedOnMemory := math.Ceil(unfulfilledMemory / maxResources.AvailableMemory)
						neededNodeBasedOnPods := math.Ceil(float64(unfulfilledPods) / float64(maxResources.AvailablePods))

						log.Infof(
							"[EventCronJob] Event : %s, Node pool %s, %f unfulfilled cpu (need %f node), %f unfulfilled memory (need %f node), %d unfulfilled pods (need %f node)",
							e.Name,
							nodePoolObj.Name,
							unfulfilledCPU,
							neededNodeBasedOnCPU,
							unfulfilledMemory,
							neededNodeBasedOnMemory,
							unfulfilledPods,
							neededNodeBasedOnPods,
						)

						autoscalingData := nodePoolObj.Autoscaling

						maxNeededNode := int32(
							math.Max(
								neededNodeBasedOnCPU,
								math.Max(neededNodeBasedOnMemory, neededNodeBasedOnPods),
							),
						)

						newMaxNode := autoscalingData.MaxNodeCount

						if maxNeededNode > 0 || int32(maxResources.CurrentNodeCount) == nodePoolObj.Autoscaling.MaxNodeCount {
							newMaxNode += maxNeededNode + 5
						}

						updatedNodePool.MaxNode = newMaxNode

						updateNodePoolLock.Lock()
						defer updateNodePoolLock.Unlock()
						log.Infof(
							"[EventCronJob] Event : %s, Updating GCP node pool %s with new max node size %d (before : %d)",
							e.Name,
							nodePoolObj.Name,
							newMaxNode,
							autoscalingData.MaxNodeCount,
						)

						autoscalingData.MaxNodeCount = newMaxNode

						opData, err := c.gcpClusterUC.SetNodePoolAutoscaling(
							ctx,
							googleContainerClient,
							project,
							location,
							name,
							nodePoolObj.Name,
							autoscalingData,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							return err
						}
						op := opData.OperationData
						for {
							opData, err = c.gcpClusterUC.GetOperation(
								ctx,
								googleContainerClient,
								project,
								location,
								op.Name,
							)
							op = opData.OperationData
							if err != nil {
								return err
							}
							if op.Status == container.Operation_DONE {
								if op.Error != nil {
									return errors.New(op.Error.String())
								}
								return nil
							}
							time.Sleep(100 * time.Millisecond)
						}
					}
				}(requestedResourceData, maxResourceData, nodePool, updatedNodePools[idx]),
			)
		}

		if err := errGroup.Wait(); err != nil {
			c.handleExecEventError(db, e, err.Error())
			return
		}
	}

	if err := db.Create(&updatedNodePools).Error; err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	// Update K8s HPA
	log.Infof("[EventCronJob] Event : %s, Updating K8s HPA with new configuration", e.Name)
	err = c.clusterUC.UpdateHPAK8sObjectBatch(ctx, kubernetesClient, clusterID, selectedK8sHPAs)
	if err != nil {
		c.handleExecEventError(db, e, err.Error())
		return
	}

	for _, existingModifiedHPA := range existingModifiedHPAs {
		err := c.scheduledHPAConfigUC.UpdateScheduledHPAConfigStatusMessage(
			db,
			existingModifiedHPA.ID,
			model.HPAUpdateSuccess,
			"",
		)
		if err != nil {
			c.handleExecEventError(
				db, e, fmt.Sprintf(
					"Error Update HPA %s Namespace %s : %s", existingModifiedHPA.Name,
					existingModifiedHPA.Namespace,
					err.Error(),
				),
			)
			return
		}
	}

	e.Status = model.EventPrescaled

	err = c.eventUC.UpdateEvent(db, e)
	if err != nil {
		log.Errorf("[EventCronJob] Error Update Event : %s", err.Error())
	}

	log.Infof("[EventCronJob] Event : %s, Done executing update and calculation", e.Name)
}
