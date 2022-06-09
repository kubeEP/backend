package response

import "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"

type Base struct {
	Status constant.ResponseStatus `json:"status"`
	Code   int                     `json:"code,omitempty"`
	Data   interface{}             `json:"data"`
}
