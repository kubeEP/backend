package handler

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/response"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
	"gorm.io/gorm"
)

type Cluster interface {
	GetAllRegisteredClusters(c *fiber.Ctx) error
	GetClusterAllHPA(c *fiber.Ctx) error
	GetClusterSimpleData(c *fiber.Ctx) error
}

type cluster struct {
	kubernetesBaseHandler
	validatorInst       *validator.Validate
	generalDatacenterUC useCase.Datacenter
	db                  *gorm.DB
}

func newClusterHandler(
	validatorInst *validator.Validate,
	db *gorm.DB,
	generalDatacenterUC useCase.Datacenter,
	kubeHandler kubernetesBaseHandler,
) Cluster {
	return &cluster{
		kubernetesBaseHandler: kubeHandler,
		validatorInst:         validatorInst,
		db:                    db,
		generalDatacenterUC:   generalDatacenterUC,
	}
}

func (ch *cluster) GetAllRegisteredClusters(c *fiber.Ctx) error {

	ctx := c.Context()
	tx := ch.db.WithContext(ctx)

	existingClusters, err := ch.generalClusterUC.GetAllClustersInLocal(tx)
	if err != nil {
		return ch.errorResponse(c, err.Error())
	}
	responses := make([]response.Cluster, 0)
	for _, cluster := range existingClusters {
		responses = append(
			responses, response.Cluster{
				ID:             &cluster.ID,
				Name:           cluster.Name,
				Datacenter:     cluster.Datacenter.Datacenter,
				DatacenterName: cluster.Datacenter.Name,
			},
		)
	}
	return ch.successResponse(c, responses)
}

func (ch *cluster) GetClusterSimpleData(c *fiber.Ctx) error {
	clusterIDStr := c.Params("cluster_id")
	clusterID, err := uuid.Parse(clusterIDStr)
	if err != nil {
		return ch.errorResponse(c, fmt.Sprintf(errorConstant.ParamInvalid, "cluster_id"))
	}

	ctx := c.Context()

	tx := ch.db.WithContext(ctx)

	clusterData, err := ch.generalClusterUC.GetClusterAndDatacenterDataByClusterID(
		tx,
		clusterID,
	)

	if err != nil {
		return ch.errorResponse(c, err.Error())
	}

	res := response.Cluster{
		Name:           clusterData.Name,
		Datacenter:     clusterData.Datacenter.Datacenter,
		DatacenterName: clusterData.Datacenter.Name,
	}

	return ch.successResponse(c, res)
}

func (ch *cluster) GetClusterAllHPA(c *fiber.Ctx) error {
	clusterIDStr := c.Params("cluster_id")
	clusterID, err := uuid.Parse(clusterIDStr)
	if err != nil {
		return ch.errorResponse(c, fmt.Sprintf(errorConstant.ParamInvalid, "cluster_id"))
	}

	ctx := c.Context()

	tx := ch.db.WithContext(ctx)

	kubernetesClient, clusterData, err := ch.getClusterKubernetesClient(
		ctx,
		tx,
		clusterID,
	)
	if err != nil {
		return ch.errorResponse(c, err.Error())
	}

	HPAs, err := ch.generalClusterUC.GetAllHPAInCluster(
		ctx,
		kubernetesClient,
		clusterID,
		clusterData.LatestHPAAPIVersion,
	)
	if err != nil {
		return ch.errorResponse(c, err.Error())
	}
	listHPA := make([]response.SimpleHPA, 0)
	for _, hpa := range HPAs {
		listHPA = append(
			listHPA, response.SimpleHPA{
				Name:            hpa.Name,
				Namespace:       hpa.Namespace,
				MinReplicas:     hpa.MinReplicas,
				MaxReplicas:     hpa.MaxReplicas,
				CurrentReplicas: hpa.CurrentReplicas,
			},
		)
	}

	return ch.successResponse(c, listHPA)
}
