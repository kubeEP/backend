package repository

import (
	container "cloud.google.com/go/container/apiv1"
	"context"
	"fmt"
	containerEntity "google.golang.org/genproto/googleapis/container/v1"
)

type GCPCluster interface {
	GetAllCluster(
		ctx context.Context,
		clusterClient *container.ClusterManagerClient,
		projectID string,
	) (
		*containerEntity.ListClustersResponse, error,
	)
	GetCluster(
		ctx context.Context,
		clusterClient *container.ClusterManagerClient,
		projectID, location, clusterName string,
	) (*containerEntity.Cluster, error)
	SetNodePoolAutoscaling(
		ctx context.Context,
		clusterClient *container.ClusterManagerClient,
		project, clusterLocation, clusterName, nodePoolName string,
		autoscalingData *containerEntity.NodePoolAutoscaling,
	) (*containerEntity.Operation, error)
	GetOperation(
		ctx context.Context,
		clusterClient *container.ClusterManagerClient,
		project, location, operationName string,
	) (*containerEntity.Operation, error)
}

type gcpCluster struct {
}

func newGcpCluster() GCPCluster {
	return &gcpCluster{}
}

func (g *gcpCluster) GetAllCluster(
	ctx context.Context,
	clusterClient *container.ClusterManagerClient,
	projectID string,
) (
	*containerEntity.ListClustersResponse, error,
) {
	return clusterClient.ListClusters(
		ctx,
		&containerEntity.ListClustersRequest{
			Parent: fmt.Sprintf("projects/%s/locations/-", projectID),
		},
	)
}

func (g *gcpCluster) GetCluster(
	ctx context.Context,
	clusterClient *container.ClusterManagerClient,
	projectID, location, clusterName string,
) (*containerEntity.Cluster, error) {
	return clusterClient.GetCluster(
		ctx, &containerEntity.GetClusterRequest{
			Name: fmt.Sprintf(
				"projects/%s/locations/%s/cluster/%s",
				projectID,
				location,
				clusterName,
			),
		},
	)
}

func (g *gcpCluster) SetNodePoolAutoscaling(
	ctx context.Context,
	clusterClient *container.ClusterManagerClient,
	project, clusterLocation, clusterName, nodePoolName string,
	autoscalingData *containerEntity.NodePoolAutoscaling,
) (*containerEntity.Operation, error) {
	return clusterClient.SetNodePoolAutoscaling(
		ctx,
		&containerEntity.SetNodePoolAutoscalingRequest{
			Name: fmt.Sprintf(
				"projects/%s/locations/%s/clusters/%s/nodePools/%s",
				project,
				clusterLocation,
				clusterName,
				nodePoolName,
			),
			Autoscaling: autoscalingData,
		},
	)
}

func (g *gcpCluster) GetOperation(
	ctx context.Context,
	clusterClient *container.ClusterManagerClient,
	project, location, operationName string,
) (*containerEntity.Operation, error) {
	return clusterClient.GetOperation(
		ctx, &containerEntity.GetOperationRequest{Name: fmt.Sprintf(
			"projects/%s/locations/%s/operations/%s",
			project,
			location,
			operationName,
		),
		},
	)
}
