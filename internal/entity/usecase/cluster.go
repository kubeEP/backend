package UCEntity

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	v1Apps "k8s.io/api/apps/v1"
	v1Core "k8s.io/api/core/v1"
)

type ClusterData struct {
	ID                  uuid.UUID
	Name                string
	Certificate         string
	ServerEndpoint      string
	Datacenter          DatacenterDetailedData
	LatestHPAAPIVersion constant.HPAVersion
}

type K8sHPAObjectData struct {
	Version   constant.HPAVersion
	HPAObject interface{}
}

type K8sDeploymentListData struct {
	DeploymentListObject *v1Apps.DeploymentList
}

type K8sNodeListData struct {
	NodeListObject *v1Core.NodeList
}

type K8sDaemonSetListData struct {
	DaemonSetListObject *v1Apps.DaemonSetList
}
