package repository

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/config"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
)

type Repositories struct {
	Cluster            Cluster
	Datacenter         Datacenter
	Event              Event
	ScheduledHPAConfig ScheduledHPAConfig
	K8sHPA             K8sHPA
	K8sNamespace       K8sNamespace
	GCPCluster         GCPCluster
	K8SDiscovery       K8SDiscovery
	K8sDeployment      K8sDeployment
	NodePoolStatus     NodePoolStatus
	UpdatedNodePool    UpdatedNodePool
	HPAStatus          HPAStatus
	K8sNode            K8sNode
	K8sDaemonSets      K8sDaemonSets
}

func Migrate(db *gorm.DB) error {
	tableList := []interface{}{
		&model.Datacenter{},
		&model.Cluster{},
		&model.Event{},
		&model.ScheduledHPAConfig{},
		&model.NodePoolStatus{},
		&model.HPAStatus{},
		&model.UpdatedNodePool{},
	}

	err := db.AutoMigrate(
		tableList...,
	)

	if err != nil {
		return err
	}

	for _, table := range tableList {
		modelObj := table.(model.Model)
		err = modelObj.AdditionalMigration(db)
		if err != nil {
			return err
		}
	}

	return nil
}

func BuildRepositories(resources *config.KubeEPResources) *Repositories {
	return &Repositories{
		Cluster:            newCluster(),
		Datacenter:         newDatacenter(resources.Redis),
		Event:              newEvent(),
		ScheduledHPAConfig: newScheduledHPAConfig(),
		K8sHPA:             newK8sHPA(resources.Redis),
		K8sNamespace:       newK8sNamespace(),
		GCPCluster:         newGcpCluster(),
		K8SDiscovery:       newK8sDiscovery(),
		K8sDeployment:      newK8sDeployment(),
		NodePoolStatus:     newNodePoolStatus(),
		HPAStatus:          newHpaStatus(),
		UpdatedNodePool:    newUpdatedNodePool(),
		K8sNode:            newK8sNode(),
		K8sDaemonSets:      newK8sDaemonSets(),
	}
}
