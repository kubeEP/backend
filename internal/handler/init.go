package handler

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/config"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
)

type Handlers struct {
	GcpHandler     Gcp
	ClusterHandler Cluster
	EventHandler   Event
}

func BuildHandlers(useCases *useCase.UseCases, resources *config.KubeEPResources) *Handlers {
	kubernetesBaseHandler := kubernetesBaseHandler{
		generalClusterUC: useCases.Cluster,
		gcpClusterUC:     useCases.GcpCluster,
		gcpDatacenterUC:  useCases.GcpDatacenter,
	}
	return &Handlers{
		GcpHandler: newGCPHandler(
			resources.ValidatorInst,
			useCases.GcpCluster,
			useCases.GcpDatacenter,
			resources.DB,
			useCases.Cluster,
		),
		ClusterHandler: newClusterHandler(
			resources.ValidatorInst,
			resources.DB,
			useCases.Datacenter,
			kubernetesBaseHandler,
		),
		EventHandler: newEventHandler(
			resources.ValidatorInst,
			useCases.Event,
			useCases.ScheduledHPAConfig,
			useCases.UpdatedNodePool,
			resources.DB,
			kubernetesBaseHandler,
		),
	}

}
