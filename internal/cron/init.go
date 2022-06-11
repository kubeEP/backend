package cron

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/config"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
)

func BuildCron(useCases *useCase.UseCases, resources *config.KubeEPResources) Cron {
	return newCron(
		useCases.Event,
		useCases.Cluster,
		useCases.GcpCluster,
		useCases.GcpDatacenter,
		useCases.ScheduledHPAConfig,
		useCases.UpdatedNodePool,
		resources.DB,
	)
}
