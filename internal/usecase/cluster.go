package useCase

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	v1Apps "k8s.io/api/apps/v1"
	v1hpa "k8s.io/api/autoscaling/v1"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/api/autoscaling/v2beta2"
	v1Core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync"
)

type Cluster interface {
	GetAllClustersInLocalByDatacenterID(
		tx *gorm.DB,
		datacenterID uuid.UUID,
	) ([]UCEntity.ClusterData, error)
	GetAllClustersInLocal(tx *gorm.DB) ([]UCEntity.ClusterData, error)
	GetAllHPAInCluster(
		ctx context.Context,
		client kubernetes.Interface,
		clusterID uuid.UUID,
		latestHPAVersion constant.HPAVersion,
	) (output []UCEntity.SimpleHPAData, err error)
	GetClusterAndDatacenterDataByClusterID(tx *gorm.DB, id uuid.UUID) (*UCEntity.ClusterData, error)
	GetLatestHPAAPIVersion(
		k8sClient kubernetes.Interface,
	) (
		constant.HPAVersion,
		error,
	)
	GetAllK8sHPAObjectInCluster(
		ctx context.Context,
		client kubernetes.Interface,
		clusterID uuid.UUID,
		latestHPAVersion constant.HPAVersion,
	) (output []UCEntity.K8sHPAObjectData, err error)
	UpdateHPAK8sObjectBatch(
		ctx context.Context,
		client kubernetes.Interface,
		clusterID uuid.UUID,
		hpaObjectList []interface{},
	) error
	ResolveScaleTargetRef(
		ctx context.Context,
		client kubernetes.Interface,
		scaleTargetRef interface{},
		namespace string,
	) (res interface{}, err error)
	ResolveScaleTargetRefByDeploymentsMap(
		scaleTargetRef interface{},
		namespace string,
		deploymentsMap map[string]v1Apps.Deployment,
		deleteKey ...bool,
	) (res *v1Apps.Deployment, err error)
	GetAllDeployments(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
	) (*UCEntity.K8sDeploymentListData, error)
	GetAllDaemonSetsInNamespace(
		ctx context.Context,
		client kubernetes.Interface,
		namespace string,
	) (*UCEntity.K8sDaemonSetListData, error)
}

type cluster struct {
	validatorInst  *validator.Validate
	clusterRepo    repository.Cluster
	hpaRepo        repository.K8sHPA
	namespaceRepo  repository.K8sNamespace
	deploymentRepo repository.K8sDeployment
	discoveryRepo  repository.K8SDiscovery
	daemonSetRepo  repository.K8sDaemonSets
}

func newCluster(
	validatorInst *validator.Validate,
	clusterRepo repository.Cluster,
	hpaRepo repository.K8sHPA,
	namespaceRepo repository.K8sNamespace,
	discoveryRepo repository.K8SDiscovery,
	deploymentRepo repository.K8sDeployment,
	daemonSetRepo repository.K8sDaemonSets,
) Cluster {
	return &cluster{
		validatorInst:  validatorInst,
		clusterRepo:    clusterRepo,
		hpaRepo:        hpaRepo,
		namespaceRepo:  namespaceRepo,
		discoveryRepo:  discoveryRepo,
		deploymentRepo: deploymentRepo,
		daemonSetRepo:  daemonSetRepo,
	}
}

func (c *cluster) GetLatestHPAAPIVersion(
	k8sClient kubernetes.Interface,
) (
	constant.HPAVersion,
	error,
) {
	response, err := c.discoveryRepo.GetServerGroups(k8sClient)
	if err != nil {
		return "", err
	}
	var autoscalingAPIGroup v1.APIGroup
	for _, apiGroup := range response.Groups {
		if apiGroup.Name == "autoscaling" {
			autoscalingAPIGroup = apiGroup
			break
		}
	}
	versionCount := len(autoscalingAPIGroup.Versions)
	latestVersion := autoscalingAPIGroup.Versions[versionCount-1]
	return latestVersion.GroupVersion, nil
}

func (c *cluster) GetAllClustersInLocalByDatacenterID(
	tx *gorm.DB,
	datacenterID uuid.UUID,
) ([]UCEntity.ClusterData, error) {
	clusters, err := c.clusterRepo.ListClusterByDatacenterID(tx, datacenterID)
	if err != nil {
		return nil, err
	}
	var output []UCEntity.ClusterData
	for _, cluster := range clusters {
		output = append(
			output, UCEntity.ClusterData{
				ID:             cluster.ID.GetUUID(),
				Name:           cluster.Name,
				Certificate:    cluster.Certificate,
				ServerEndpoint: cluster.ServerEndpoint,
				Datacenter: UCEntity.DatacenterDetailedData{
					Datacenter: cluster.Datacenter.Datacenter,
				},
			},
		)
	}
	return output, nil
}

func (c *cluster) GetAllClustersInLocal(tx *gorm.DB) ([]UCEntity.ClusterData, error) {
	clusters, err := c.clusterRepo.ListAllRegisteredCluster(tx)
	if err != nil {
		return nil, err
	}
	var output []UCEntity.ClusterData
	for _, cluster := range clusters {
		output = append(
			output, UCEntity.ClusterData{
				ID:             cluster.ID.GetUUID(),
				Name:           cluster.Name,
				Certificate:    cluster.Certificate,
				ServerEndpoint: cluster.ServerEndpoint,
				Datacenter: UCEntity.DatacenterDetailedData{
					Datacenter: cluster.Datacenter.Datacenter,
					Name:       cluster.Datacenter.Name,
				},
			},
		)
	}
	return output, nil
}

func (c *cluster) GetClusterAndDatacenterDataByClusterID(
	tx *gorm.DB,
	id uuid.UUID,
) (*UCEntity.ClusterData, error) {
	data, err := c.clusterRepo.GetClusterWithDatacenterByID(tx, id)
	if err != nil {
		return nil, err
	}
	datacenterModelData := data.Datacenter
	clusterData := &UCEntity.ClusterData{
		ID:             data.ID.GetUUID(),
		Name:           data.Name,
		Certificate:    data.Certificate,
		ServerEndpoint: data.ServerEndpoint,
		Datacenter: UCEntity.DatacenterDetailedData{
			ID:          datacenterModelData.ID.GetUUID(),
			Name:        datacenterModelData.Name,
			Credentials: datacenterModelData.Credentials.GetRawMessage(),
			Metadata:    datacenterModelData.Metadata.GetRawMessage(),
			Datacenter:  datacenterModelData.Datacenter,
		},
		LatestHPAAPIVersion: data.LatestHPAAPIVersion,
	}
	return clusterData, nil
}

func (c *cluster) GetAllHPAInCluster(
	ctx context.Context,
	client kubernetes.Interface,
	clusterID uuid.UUID,
	latestHPAVersion constant.HPAVersion,
) (output []UCEntity.SimpleHPAData, err error) {
	namespaces, err := c.namespaceRepo.GetAllNamespace(ctx, client)
	if err != nil {
		return nil, err
	}
	HPAs := map[string]interface{}{}
	var lock sync.Mutex
	eg, _ := errgroup.WithContext(ctx)

	for _, namespace := range namespaces {
		loadFunc := func(ns v1Core.Namespace) func() error {
			return func() error {

				var response interface{}
				var err error

				switch latestHPAVersion {
				case constant.AutoscalingV1:
					response, err = c.hpaRepo.GetAllV1HPA(ctx, client, ns, clusterID)
				case constant.AutoscalingV2Beta1:
					response, err = c.hpaRepo.GetAllV2beta1HPA(ctx, client, ns, clusterID)
				case constant.AutoscalingV2Beta2:
					response, err = c.hpaRepo.GetAllV2beta2HPA(ctx, client, ns, clusterID)
				default:
					return errors.New(errorConstant.HPAVersionUnknown)
				}
				if err != nil {
					return err
				}
				lock.Lock()
				HPAs[ns.Name] = response
				lock.Unlock()

				return nil
			}
		}
		eg.Go(loadFunc(namespace))
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	for _, ns := range namespaces {
		v := HPAs[ns.Name]
		switch chosenHPAs := v.(type) {
		case []v1hpa.HorizontalPodAutoscaler:
			for _, hpa := range chosenHPAs {
				output = append(
					output, UCEntity.SimpleHPAData{
						Name:            hpa.Name,
						Namespace:       ns.Name,
						MinReplicas:     hpa.Spec.MinReplicas,
						MaxReplicas:     hpa.Spec.MaxReplicas,
						CurrentReplicas: hpa.Status.CurrentReplicas,
						ScaleTargetRef: UCEntity.HPAScaleTargetRef{
							Name: hpa.Spec.ScaleTargetRef.Name,
							Kind: hpa.Spec.ScaleTargetRef.Kind,
						},
					},
				)
			}
		case []v2beta1.HorizontalPodAutoscaler:
			for _, hpa := range chosenHPAs {
				output = append(
					output, UCEntity.SimpleHPAData{
						Name:            hpa.Name,
						Namespace:       ns.Name,
						MinReplicas:     hpa.Spec.MinReplicas,
						MaxReplicas:     hpa.Spec.MaxReplicas,
						CurrentReplicas: hpa.Status.CurrentReplicas,
						ScaleTargetRef: UCEntity.HPAScaleTargetRef{
							Name: hpa.Spec.ScaleTargetRef.Name,
							Kind: hpa.Spec.ScaleTargetRef.Kind,
						},
					},
				)
			}
		case []v2beta2.HorizontalPodAutoscaler:
			for _, hpa := range chosenHPAs {
				output = append(
					output, UCEntity.SimpleHPAData{
						Name:            hpa.Name,
						Namespace:       ns.Name,
						MinReplicas:     hpa.Spec.MinReplicas,
						MaxReplicas:     hpa.Spec.MaxReplicas,
						CurrentReplicas: hpa.Status.CurrentReplicas,
						ScaleTargetRef: UCEntity.HPAScaleTargetRef{
							Name: hpa.Spec.ScaleTargetRef.Name,
							Kind: hpa.Spec.ScaleTargetRef.Kind,
						},
					},
				)
			}
		}
	}
	return output, nil
}

func (c *cluster) GetAllK8sHPAObjectInCluster(
	ctx context.Context,
	client kubernetes.Interface,
	clusterID uuid.UUID,
	latestHPAVersion constant.HPAVersion,
) (output []UCEntity.K8sHPAObjectData, err error) {
	namespaces, err := c.namespaceRepo.GetAllNamespace(ctx, client)
	if err != nil {
		return nil, err
	}
	var lock sync.Mutex
	eg, ctxEg := errgroup.WithContext(ctx)

	for _, namespace := range namespaces {
		loadFunc := func(ns v1Core.Namespace) func() error {
			return func() error {
				switch latestHPAVersion {
				case constant.AutoscalingV1:
					response, err := c.hpaRepo.GetAllV1HPA(ctxEg, client, ns, clusterID)
					if err != nil {
						if ctxEg.Err() != nil {
							return nil
						}
						return err
					}
					lock.Lock()
					for _, hO := range response {
						output = append(
							output,
							UCEntity.K8sHPAObjectData{
								Version:   constant.AutoscalingV1,
								HPAObject: hO,
							},
						)
					}
					lock.Unlock()
				case constant.AutoscalingV2Beta1:
					response, err := c.hpaRepo.GetAllV2beta1HPA(ctxEg, client, ns, clusterID)
					if err != nil {
						if ctxEg.Err() != nil {
							return nil
						}
						return err
					}
					lock.Lock()
					for _, hO := range response {
						output = append(
							output,
							UCEntity.K8sHPAObjectData{
								Version:   constant.AutoscalingV2Beta1,
								HPAObject: hO,
							},
						)
					}
					lock.Unlock()
				case constant.AutoscalingV2Beta2:
					response, err := c.hpaRepo.GetAllV2beta2HPA(ctxEg, client, ns, clusterID)
					if err != nil {
						if ctxEg.Err() != nil {
							return nil
						}
						return err
					}
					lock.Lock()
					for _, hO := range response {
						output = append(
							output,
							UCEntity.K8sHPAObjectData{
								Version:   constant.AutoscalingV2Beta2,
								HPAObject: hO,
							},
						)
					}
					lock.Unlock()
				default:
					return errors.New(errorConstant.HPAVersionUnknown)
				}
				return nil
			}
		}
		eg.Go(loadFunc(namespace))
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return output, nil
}

func (c *cluster) GetAllK8sHPAObjectInList(
	ctx context.Context,
	client kubernetes.Interface,
	clusterID uuid.UUID,
	hpaList []UCEntity.SimpleHPAData,
	latestHPAVersion constant.HPAVersion,
) ([]interface{}, error) {
	errGroup, ctxEg := errgroup.WithContext(ctx)
	var lock sync.Mutex
	hpaObjectList := make([]interface{}, len(hpaList))
	for idx, hpa := range hpaList {
		errGroup.Go(
			func(h UCEntity.SimpleHPAData, i int) func() error {
				return func() error {
					var object interface{}
					var err error
					switch latestHPAVersion {
					case constant.AutoscalingV1:
						object, err = c.hpaRepo.GetV1HPA(
							ctxEg,
							client,
							h.Name,
							h.Namespace,
							clusterID,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							return err
						}
					case constant.AutoscalingV2Beta1:
						object, err = c.hpaRepo.GetV2beta1HPA(
							ctxEg,
							client,
							h.Name,
							h.Namespace,
							clusterID,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							return err
						}
					case constant.AutoscalingV2Beta2:
						object, err = c.hpaRepo.GetV2beta2HPA(
							ctxEg,
							client,
							h.Name,
							h.Namespace,
							clusterID,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
							return err
						}
					}
					lock.Lock()
					hpaObjectList[i] = object
					lock.Unlock()

					return nil
				}
			}(hpa, idx),
		)
	}

	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	return hpaObjectList, nil
}

func (c *cluster) UpdateHPAK8sObjectBatch(
	ctx context.Context,
	client kubernetes.Interface,
	clusterID uuid.UUID,
	hpaObjectList []interface{},
) error {
	errGroup, ctxEg := errgroup.WithContext(ctx)
	for _, hpaObject := range hpaObjectList {
		errGroup.Go(
			func(hO interface{}) func() error {
				return func() error {
					switch h := hO.(type) {
					case *v1hpa.HorizontalPodAutoscaler:
						h.ResourceVersion = ""
						_, err := c.hpaRepo.UpdateV1HPA(ctxEg, client, h.Namespace, clusterID, h)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
						}
						return err
					case *v2beta1.HorizontalPodAutoscaler:
						h.ResourceVersion = ""
						_, err := c.hpaRepo.UpdateV2beta1HPA(
							ctxEg,
							client,
							h.Namespace,
							clusterID,
							h,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
						}
						return err
					case *v2beta2.HorizontalPodAutoscaler:
						h.ResourceVersion = ""
						_, err := c.hpaRepo.UpdateV2beta2HPA(
							ctxEg,
							client,
							h.Namespace,
							clusterID,
							h,
						)
						if err != nil {
							if ctxEg.Err() != nil {
								return nil
							}
						}
						return err
					default:
						return errors.New("unknown type")
					}
				}
			}(hpaObject),
		)
	}
	return errGroup.Wait()
}

func (c *cluster) ResolveScaleTargetRefByDeploymentsMap(
	scaleTargetRef interface{},
	namespace string,
	deploymentsMap map[string]v1Apps.Deployment,
	deleteKey ...bool,
) (res *v1Apps.Deployment, err error) {
	var apiVersion, kind, name string

	switch ref := scaleTargetRef.(type) {
	case v1hpa.CrossVersionObjectReference:
		apiVersion = ref.APIVersion
		kind = ref.Kind
		name = ref.Name
	case v2beta1.CrossVersionObjectReference:
		apiVersion = ref.APIVersion
		kind = ref.Kind
		name = ref.Name
	case v2beta2.CrossVersionObjectReference:
		apiVersion = ref.APIVersion
		kind = ref.Kind
		name = ref.Name
	default:
		return nil, errors.New(errorConstant.HPAVersionUnknown)
	}

	switch apiVersion {
	case constant.AppsV1:
	default:
		return nil, errors.New(errorConstant.TargetRefResolveError)
	}

	switch kind {
	case constant.Deployment:
		key := fmt.Sprintf(constant.NameNSKeyFormat, name, namespace)
		res, ok := deploymentsMap[key]
		if !ok {
			return nil, errors.New(errorConstant.DeploymentNotFound)
		}
		if len(deleteKey) > 0 {
			delete(deploymentsMap, key)
		}
		return &res, nil
	default:
		return nil, errors.New(errorConstant.TargetRefResolveError)
	}
}

func (c *cluster) ResolveScaleTargetRef(
	ctx context.Context,
	client kubernetes.Interface,
	scaleTargetRef interface{},
	namespace string,
) (res interface{}, err error) {
	var apiVersion, kind, name string

	switch ref := scaleTargetRef.(type) {
	case v1hpa.CrossVersionObjectReference:
		apiVersion = ref.APIVersion
		kind = ref.Kind
		name = ref.Name
	case v2beta1.CrossVersionObjectReference:
		apiVersion = ref.APIVersion
		kind = ref.Kind
		name = ref.Name
	case v2beta2.CrossVersionObjectReference:
		apiVersion = ref.APIVersion
		kind = ref.Kind
		name = ref.Name
	default:
		return nil, errors.New(errorConstant.HPAVersionUnknown)
	}

	switch apiVersion {
	case constant.AppsV1:
	default:
		return nil, errors.New(errorConstant.TargetRefResolveError)
	}

	switch kind {
	case constant.Deployment:
		res, err = c.deploymentRepo.GetDeployment(ctx, client, namespace, name)
	default:
		return nil, errors.New(errorConstant.TargetRefResolveError)
	}
	return
}

func (c *cluster) GetAllDeployments(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
) (*UCEntity.K8sDeploymentListData, error) {
	data, err := c.deploymentRepo.GetAllDeployment(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	return &UCEntity.K8sDeploymentListData{DeploymentListObject: data}, nil
}

func (c *cluster) GetAllDaemonSetsInNamespace(
	ctx context.Context,
	client kubernetes.Interface,
	namespace string,
) (*UCEntity.K8sDaemonSetListData, error) {
	data, err := c.daemonSetRepo.GetDaemonSetsList(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	return &UCEntity.K8sDaemonSetListData{DaemonSetListObject: data}, nil
}
