package UCEntity

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
)

type DatacenterData struct {
	Credentials json.RawMessage
	Name        string
}

type DatacenterDetailedData struct {
	ID          uuid.UUID
	Name        string
	Credentials json.RawMessage
	Metadata    json.RawMessage
	Datacenter  model.DatacenterProvider
}
