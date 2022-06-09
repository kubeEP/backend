package UCEntity

import (
	"google.golang.org/genproto/googleapis/container/v1"
)

type GCPClusterData struct {
	ClusterData
	Location string
}

type GCPClusterMetaData struct {
	Location string `json:"location"`
}

type GCPClusterObjectData struct {
	ClusterObject *container.Cluster
}

type GCPClusterOperationData struct {
	OperationData *container.Operation
}
