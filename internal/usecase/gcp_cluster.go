package useCase

import (
	compute "cloud.google.com/go/compute/apiv1"
	container "cloud.google.com/go/container/apiv1"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	gcpCustomAuth "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/k8s/auth/gcp_custom"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/k8s/client"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	containerEntity "google.golang.org/genproto/googleapis/container/v1"
	"gorm.io/gorm"
	v1Option "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
)

type GCPCluster interface {
	RegisterGoogleCredentials(credentialsName string, gcpCredentials *google.Credentials)
	GetAllClustersInGCPProject(
		ctx context.Context,
		projectID string,
		clusterClient *container.ClusterManagerClient,
	) ([]*UCEntity.GCPClusterData, error)
	GetGoogleClusterClient(
		ctx context.Context,
		googleCredential *google.Credentials,
	) (*container.ClusterManagerClient, error)
	RegisterClusters(
		tx *gorm.DB,
		datacenterID uuid.UUID,
		listCluster []*UCEntity.GCPClusterData,
	) error
	GetKubernetesClusterClient(
		credentialsName string,
		clusterData *UCEntity.ClusterData,
	) (*kubernetes.Clientset, error)
	GetGoogleInstanceGroupManagersClient(
		ctx context.Context,
		googleCredential *google.Credentials,
	) (*compute.InstanceGroupManagersClient, error)
	GetGoogleInstanceTemplatesClient(
		ctx context.Context,
		googleCredential *google.Credentials,
	) (*compute.InstanceTemplatesClient, error)
	GetGCPClusterObject(
		ctx context.Context,
		clusterClient *container.ClusterManagerClient,
		projectID, location, clusterName string,
	) (*UCEntity.GCPClusterObjectData, error)
	GetNodesFromGCPNodePool(
		ctx context.Context,
		k8sClient kubernetes.Interface,
		nodePoolName string,
	) (*UCEntity.K8sNodeListData, error)
	SetNodePoolAutoscaling(
		ctx context.Context, clusterClient *container.ClusterManagerClient,
		project, location, clusterName, nodePoolName string,
		autoscalingData *containerEntity.NodePoolAutoscaling,
	) (*UCEntity.GCPClusterOperationData, error)
	GetOperation(
		ctx context.Context,
		clusterClient *container.ClusterManagerClient,
		project, location, operationName string,
	) (*UCEntity.GCPClusterOperationData, error)
}

type gcpCluster struct {
	validatorInst    *validator.Validate
	clusterRepo      repository.Cluster
	gcpClusterRepo   repository.GCPCluster
	k8sDiscoveryRepo repository.K8SDiscovery
	k8sNodeRepo      repository.K8sNode
}

func newGCPCluster(
	validatorInst *validator.Validate,
	clusterRepo repository.Cluster,
	gcpClusterRepo repository.GCPCluster,
	k8sDiscoveryRepo repository.K8SDiscovery,
	k8sNodeRepo repository.K8sNode,
) GCPCluster {
	return &gcpCluster{
		validatorInst:    validatorInst,
		clusterRepo:      clusterRepo,
		gcpClusterRepo:   gcpClusterRepo,
		k8sDiscoveryRepo: k8sDiscoveryRepo,
		k8sNodeRepo:      k8sNodeRepo,
	}
}

func (c *gcpCluster) RegisterGoogleCredentials(
	credentialsName string,
	gcpCredentials *google.Credentials,
) {
	gcpCustomAuth.RegisterGoogleCredentials(credentialsName, gcpCredentials)
}

func (c *gcpCluster) GetGoogleClusterClient(
	ctx context.Context,
	googleCredential *google.Credentials,
) (*container.ClusterManagerClient, error) {
	return container.NewClusterManagerClient(ctx, option.WithCredentials(googleCredential))
}

func (c *gcpCluster) GetGoogleInstanceGroupManagersClient(
	ctx context.Context,
	googleCredential *google.Credentials,
) (*compute.InstanceGroupManagersClient, error) {
	return compute.NewInstanceGroupManagersRESTClient(ctx, option.WithCredentials(googleCredential))
}

func (c *gcpCluster) GetGoogleInstanceTemplatesClient(
	ctx context.Context,
	googleCredential *google.Credentials,
) (*compute.InstanceTemplatesClient, error) {
	return compute.NewInstanceTemplatesRESTClient(ctx, option.WithCredentials(googleCredential))
}

func (c *gcpCluster) GetAllClustersInGCPProject(
	ctx context.Context,
	projectID string,
	clusterClient *container.ClusterManagerClient,
) ([]*UCEntity.GCPClusterData, error) {
	clusters, err := c.gcpClusterRepo.GetAllCluster(ctx, clusterClient, projectID)
	if err != nil {
		return nil, err
	}
	var clusterData []*UCEntity.GCPClusterData
	for _, cluster := range clusters.GetClusters() {
		clusterData = append(
			clusterData, &UCEntity.GCPClusterData{
				ClusterData: UCEntity.ClusterData{
					Name: fmt.Sprintf(
						"gke_%s_%s_%s",
						projectID,
						cluster.GetName(),
						cluster.GetLocation(),
					),
					Certificate:    cluster.GetMasterAuth().GetClusterCaCertificate(),
					ServerEndpoint: fmt.Sprintf("https://%s", cluster.GetEndpoint()),
					Datacenter: UCEntity.DatacenterDetailedData{
						Datacenter: model.GCP,
					},
				},
				Location: cluster.GetLocation(),
			},
		)
	}
	return clusterData, nil
}

func (c *gcpCluster) RegisterClusters(
	tx *gorm.DB,
	datacenterID uuid.UUID,
	listCluster []*UCEntity.GCPClusterData,
) error {
	var clusters []*model.Cluster
	for _, cluster := range listCluster {
		metadata := UCEntity.GCPClusterMetaData{Location: cluster.Location}
		metadataByte, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		clusterModel := &model.Cluster{
			Name:                cluster.Name,
			ServerEndpoint:      cluster.ServerEndpoint,
			Certificate:         cluster.Certificate,
			LatestHPAAPIVersion: cluster.LatestHPAAPIVersion,
		}
		clusterModel.DatacenterID.SetUUID(datacenterID)
		clusterModel.Metadata.SetRawMessage(metadataByte)
		clusters = append(clusters, clusterModel)
	}

	err := c.clusterRepo.InsertClusterBatch(tx, clusters)
	if err != nil {
		return err
	}

	for idx, cluster := range clusters {
		listCluster[idx].ID = cluster.ID.GetUUID()
	}

	return nil
}

func (c *gcpCluster) GetGCPClusterObject(
	ctx context.Context,
	clusterClient *container.ClusterManagerClient,
	projectID, location, clusterName string,
) (*UCEntity.GCPClusterObjectData, error) {
	clusterData, err := c.gcpClusterRepo.GetCluster(
		ctx,
		clusterClient,
		projectID,
		location,
		clusterName,
	)
	if err != nil {
		return nil, err
	}
	return &UCEntity.GCPClusterObjectData{ClusterObject: clusterData}, nil
}

func (c *gcpCluster) GetKubernetesClusterClient(
	credentialsName string,
	clusterData *UCEntity.ClusterData,
) (*kubernetes.Clientset, error) {
	if clusterData.Datacenter.Datacenter != model.GCP {
		return nil, errors.New(errorConstant.DatacenterMismatch)
	}

	credentials := &k8sClient.Credentials{
		Certificate:    clusterData.Certificate,
		Name:           clusterData.Name,
		ServerEndpoint: clusterData.ServerEndpoint,
		AuthProviderConfig: &api.AuthProviderConfig{
			Name: gcpCustomAuth.AuthName,
			Config: map[string]string{
				gcpCustomAuth.CredentialsNameConfigKey: credentialsName,
			},
		},
	}

	return k8sClient.GetClient(credentials)
}

func (c *gcpCluster) GetNodesFromGCPNodePool(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	nodePoolName string,
) (*UCEntity.K8sNodeListData, error) {
	data, err := c.k8sNodeRepo.GetNodeList(
		ctx, k8sClient, v1Option.ListOptions{
			LabelSelector: fmt.Sprintf(
				"%s=%s",
				constant.GCPNodePoolLabel,
				nodePoolName,
			),
		},
	)
	if err != nil {
		return nil, err
	}
	if len(data.Items) == 0 {
		return nil, errors.New(errorConstant.NoExistingNode)
	}
	return &UCEntity.K8sNodeListData{NodeListObject: data}, nil
}

func (c *gcpCluster) SetNodePoolAutoscaling(
	ctx context.Context, clusterClient *container.ClusterManagerClient,
	project, location, clusterName, nodePoolName string,
	autoscalingData *containerEntity.NodePoolAutoscaling,
) (*UCEntity.GCPClusterOperationData, error) {
	op, err := c.gcpClusterRepo.SetNodePoolAutoscaling(
		ctx,
		clusterClient,
		project,
		location,
		clusterName,
		nodePoolName,
		autoscalingData,
	)
	if err != nil {
		return nil, err
	}
	return &UCEntity.GCPClusterOperationData{OperationData: op}, nil
}

func (c *gcpCluster) GetOperation(
	ctx context.Context,
	clusterClient *container.ClusterManagerClient,
	project, location, operationName string,
) (*UCEntity.GCPClusterOperationData, error) {
	op, err := c.gcpClusterRepo.GetOperation(ctx, clusterClient, project, location, operationName)
	if err != nil {
		return nil, err
	}
	return &UCEntity.GCPClusterOperationData{OperationData: op}, nil
}
