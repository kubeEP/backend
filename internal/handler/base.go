package handler

import (
	"context"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/response"
	UCEntity "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type baseHandler struct {
}

func (h baseHandler) errorResponse(c *fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusBadRequest).JSON(
		&response.Base{
			Status: constant.Error,
			Data:   data,
		},
	)
}

func (h baseHandler) errorResponseWithCode(c *fiber.Ctx, data interface{}, code int) error {
	return c.Status(http.StatusBadRequest).JSON(
		&response.Base{
			Code:   code,
			Status: constant.Error,
			Data:   data,
		},
	)
}

func (h baseHandler) successResponse(c *fiber.Ctx, data interface{}) error {
	return c.JSON(
		&response.Base{
			Data:   data,
			Status: constant.Success,
		},
	)
}

func (h baseHandler) failResponse(c *fiber.Ctx, data interface{}) error {
	return c.JSON(
		&response.Base{
			Data:   data,
			Status: constant.Fail,
		},
	)
}

type kubernetesBaseHandler struct {
	baseHandler
	generalClusterUC useCase.Cluster
	gcpClusterUC     useCase.GCPCluster
	gcpDatacenterUC  useCase.GCPDatacenter
}

func (h kubernetesBaseHandler) getClusterKubernetesClient(
	ctx context.Context,
	tx *gorm.DB,
	clusterID uuid.UUID,
) (kubernetes.Interface, *UCEntity.ClusterData, error) {
	clusterData, err := h.generalClusterUC.GetClusterAndDatacenterDataByClusterID(
		tx,
		clusterID,
	)
	if err != nil {
		return nil, nil, err
	}
	var kubernetesClient kubernetes.Interface
	switch clusterData.Datacenter.Datacenter {
	case model.GCP:
		datacenterName := clusterData.Datacenter.Name
		datacenterData := UCEntity.DatacenterData{
			Credentials: clusterData.Datacenter.Credentials,
			Name:        datacenterName,
		}
		googleCredential, err := h.gcpDatacenterUC.GetGoogleCredentials(
			ctx,
			datacenterData,
		)
		if err != nil {
			return nil, nil, err
		}
		h.gcpClusterUC.RegisterGoogleCredentials(datacenterName, googleCredential)
		kubernetesClient, err = h.gcpClusterUC.GetKubernetesClusterClient(
			datacenterName,
			clusterData,
		)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, errors.New(errorConstant.DatacenterTypeNotFound)
	}
	return kubernetesClient, clusterData, nil
}
