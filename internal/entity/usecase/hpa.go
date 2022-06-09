package UCEntity

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
)

type HPAScaleTargetRef struct {
	Name string
	Kind string
}

type SimpleHPAData struct {
	Name            string
	Namespace       string
	MinReplicas     *int32
	MaxReplicas     int32
	CurrentReplicas int32
	ScaleTargetRef  HPAScaleTargetRef
}

type EventModifiedHPAConfigData struct {
	ID          uuid.UUID
	Name        string
	Namespace   string
	Status      model.HPAUpdateStatus
	Message     string
	MinReplicas *int32
	MaxReplicas int32
}
