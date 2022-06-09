package useCase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/contactcenterinsights/v1"
	"gorm.io/gorm"
	"time"
)

type GCPDatacenter interface {
	SaveDatacenterDetailedData(tx *gorm.DB, data *UCEntity.DatacenterDetailedData) (
		uuid.UUID,
		error,
	)
	SaveDatacenter(
		tx *gorm.DB,
		data UCEntity.DatacenterData,
		SACredentials *UCEntity.GCPSAKeyCredentials,
	) (uuid.UUID, error)
	ParseServiceAccountKey(data UCEntity.DatacenterData) (*UCEntity.GCPSAKeyCredentials, error)
	GetGoogleCredentials(ctx context.Context, data UCEntity.DatacenterData) (
		*google.Credentials,
		error,
	)
	SaveTemporaryDatacenter(
		ctx context.Context,
		data UCEntity.DatacenterData,
		SACredentials *UCEntity.GCPSAKeyCredentials,
	) (uuid.UUID, error)
	GetTemporaryDatacenterData(ctx context.Context, id uuid.UUID) (
		*UCEntity.DatacenterDetailedData,
		error,
	)
	GetDatacenterData(tx *gorm.DB, id uuid.UUID) (*UCEntity.DatacenterDetailedData, error)
}

type gcpDatacenter struct {
	datacenterRepo repository.Datacenter
	validatorInst  *validator.Validate
}

func newGCPDatacenter(
	datacenterRepo repository.Datacenter,
	validatorInst *validator.Validate,
) GCPDatacenter {
	return &gcpDatacenter{
		datacenterRepo: datacenterRepo,
		validatorInst:  validatorInst,
	}
}

func (d *gcpDatacenter) ParseServiceAccountKey(data UCEntity.DatacenterData) (
	*UCEntity.GCPSAKeyCredentials,
	error,
) {
	SACredentials := &UCEntity.GCPSAKeyCredentials{}
	err := json.Unmarshal(data.Credentials, SACredentials)
	if err != nil {
		return nil, err
	}
	err = d.validatorInst.Struct(SACredentials)
	if err != nil {
		return nil, errors.New(errorConstant.SAKeyInvalid)
	}
	return SACredentials, nil
}

func (d *gcpDatacenter) SaveTemporaryDatacenter(
	ctx context.Context,
	data UCEntity.DatacenterData,
	SACredentials *UCEntity.GCPSAKeyCredentials,
) (uuid.UUID, error) {
	metaData := &UCEntity.GCPDatacenterMetaData{
		ProjectId: *SACredentials.ProjectId,
		SAEmail:   *SACredentials.ClientEmail,
	}
	metaDataByte, err := json.Marshal(metaData)
	if err != nil {
		return uuid.UUID{}, err
	}
	datacenterModel := &model.Datacenter{
		Name:       data.Name,
		Datacenter: model.GCP,
	}
	datacenterModel.Credentials.SetRawMessage(data.Credentials)
	datacenterModel.Metadata.SetRawMessage(metaDataByte)
	err = d.datacenterRepo.InsertTemporaryDatacenter(ctx, datacenterModel, time.Hour)
	return datacenterModel.ID.GetUUID(), err
}

func (d *gcpDatacenter) GetTemporaryDatacenterData(
	ctx context.Context,
	id uuid.UUID,
) (*UCEntity.DatacenterDetailedData, error) {
	data, err := d.datacenterRepo.GetTemporaryDatacenterByID(ctx, id)
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

func (d *gcpDatacenter) GetDatacenterData(
	tx *gorm.DB,
	id uuid.UUID,
) (*UCEntity.DatacenterDetailedData, error) {
	data, err := d.datacenterRepo.GetDatacenterByID(tx, id)
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

func (d *gcpDatacenter) SaveDatacenterDetailedData(
	tx *gorm.DB,
	data *UCEntity.DatacenterDetailedData,
) (uuid.UUID, error) {
	datacenterData := &model.Datacenter{
		Name:       data.Name,
		Datacenter: data.Datacenter,
	}
	datacenterData.ID.SetUUID(data.ID)
	datacenterData.Credentials.SetRawMessage(data.Credentials)
	datacenterData.Metadata.SetRawMessage(data.Metadata)
	err := d.datacenterRepo.InsertDatacenter(tx, datacenterData)
	return datacenterData.ID.GetUUID(), err
}

func (d *gcpDatacenter) SaveDatacenter(
	tx *gorm.DB,
	data UCEntity.DatacenterData,
	SACredentials *UCEntity.GCPSAKeyCredentials,
) (uuid.UUID, error) {
	metaData := &UCEntity.GCPDatacenterMetaData{
		ProjectId: *SACredentials.ProjectId,
		SAEmail:   *SACredentials.ClientEmail,
	}
	metaDataByte, err := json.Marshal(metaData)
	if err != nil {
		return uuid.UUID{}, err
	}
	datacenterModel := model.Datacenter{
		Name:       data.Name,
		Datacenter: model.GCP,
	}
	datacenterModel.Credentials.SetRawMessage(data.Credentials)
	datacenterModel.Metadata.SetRawMessage(metaDataByte)
	err = d.datacenterRepo.InsertDatacenter(tx, &datacenterModel)
	return uuid.UUID(datacenterModel.ID), err
}

func (d *gcpDatacenter) GetGoogleCredentials(
	ctx context.Context,
	data UCEntity.DatacenterData,
) (*google.Credentials, error) {
	credentials, err := google.CredentialsFromJSON(
		ctx,
		data.Credentials,
		contactcenterinsights.CloudPlatformScope,
	)
	if err != nil {
		return nil, err
	}
	return credentials, nil
}
