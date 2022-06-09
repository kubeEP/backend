package useCase

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"gorm.io/gorm"
)

type Datacenter interface {
	GetDatacenterByClusterID(tx *gorm.DB, clusterID uuid.UUID) (*UCEntity.DatacenterDetailedData, error)
}

type datacenter struct {
	validatorInst  *validator.Validate
	datacenterRepo repository.Datacenter
}

func newDatacenter(validatorInst *validator.Validate, datacenterRepo repository.Datacenter) Datacenter {
	return &datacenter{validatorInst: validatorInst, datacenterRepo: datacenterRepo}
}

func (d datacenter) GetDatacenterByClusterID(tx *gorm.DB, clusterID uuid.UUID) (*UCEntity.DatacenterDetailedData, error) {
	data, err := d.datacenterRepo.GetDatacenterByClusterID(tx, clusterID)
	if err != nil {
		return nil, err
	}
	return &UCEntity.DatacenterDetailedData{
		ID:          data.ID.GetUUID(),
		Name:        data.Name,
		Credentials: data.Credentials.GetRawMessage(),
		Metadata:    data.Metadata.GetRawMessage(),
		Datacenter:  data.Datacenter,
	}, nil
}
