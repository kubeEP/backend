package cron

import (
	compute "cloud.google.com/go/compute/apiv1"
	container "cloud.google.com/go/container/apiv1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type NodePoolRequestedResourceData struct {
	MaxCPU    float64
	MaxMemory float64
	MaxPods   int64
}

type NodePoolResourceData struct {
	MaxAvailablePods   int64
	MaxAvailableCPU    float64
	MaxAvailableMemory float64
	AvailableCPU       float64
	AvailableMemory    float64
	AvailablePods      int64
	CurrentNodeCount   int
	NodeLabels         labels.Set
}

type DeploymentPodData struct {
	Name, Namespace     string
	Replicas            int32
	AvailableReplicas   int32
	ReadyReplicas       int32
	UnavailableReplicas int32
}

type DaemonSetData struct {
	NodeSelector    labels.Selector
	NodeAffinity    *v1.NodeAffinity
	RequestedMemory float64
	RequestedCPU    float64
	Name, Namespace string
}

type GCPClients struct {
	clusterClient               *container.ClusterManagerClient
	instanceGroupManagersClient *compute.InstanceGroupManagersClient
	instanceTemplatesClient     *compute.InstanceTemplatesClient
}
