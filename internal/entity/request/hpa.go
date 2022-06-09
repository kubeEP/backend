package request

type EventModifiedHPAConfigData struct {
	Name        *string `json:"name" validate:"required"`
	Namespace   *string `json:"namespace" validate:"required"`
	MinReplicas *int32  `json:"min_replicas" validate:"required"`
	MaxReplicas *int32  `json:"max_replicas" validate:"required"`
}
