package handler

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	errorConstant "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/constant/errors"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/request"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/response"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/entity/usecase"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	useCase "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/usecase"
	"gorm.io/gorm"
)

type Gcp interface {
	RegisterDatacenter(c *fiber.Ctx) error
	GetClustersByDatacenterID(c *fiber.Ctx) error
	RegisterClusterWithDatacenter(c *fiber.Ctx) error
}

type gcp struct {
	baseHandler
	validatorInst    *validator.Validate
	clusterUC        useCase.GCPCluster
	generalClusterUC useCase.Cluster
	datacenterUC     useCase.GCPDatacenter
	db               *gorm.DB
}

func newGCPHandler(
	validatorInst *validator.Validate,
	clusterUC useCase.GCPCluster,
	datacenterUC useCase.GCPDatacenter,
	db *gorm.DB,
	generalClusterUC useCase.Cluster,
) Gcp {

	return &gcp{
		validatorInst:    validatorInst,
		clusterUC:        clusterUC,
		datacenterUC:     datacenterUC,
		generalClusterUC: generalClusterUC,
		db:               db,
	}
}

func (g *gcp) RegisterDatacenter(c *fiber.Ctx) error {
	reqData := &request.GCPDatacenterData{}
	err := c.BodyParser(reqData)
	if err != nil {
		return g.errorResponse(c, errorConstant.InvalidRequestBody)
	}
	err = g.validatorInst.Struct(reqData)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	ctx := c.Context()
	tx := g.db.WithContext(ctx)

	datacenterData := UCEntity.DatacenterData{
		Credentials: *reqData.SAKeyCredentials,
		Name:        *reqData.Name,
	}
	SAData, err := g.datacenterUC.ParseServiceAccountKey(datacenterData)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	var id uuid.UUID
	if *reqData.IsTemporary {
		id, err = g.datacenterUC.SaveTemporaryDatacenter(ctx, datacenterData, SAData)
	} else {
		id, err = g.datacenterUC.SaveDatacenter(tx, datacenterData, SAData)
	}

	return g.successResponse(
		c,
		response.GCPDatacenterData{DatacenterID: id, IsTemporary: *reqData.IsTemporary},
	)
}

func (g *gcp) GetClustersByDatacenterID(c *fiber.Ctx) error {
	reqData := &request.GCPExistingDatacenterData{}
	err := c.QueryParser(reqData)
	if err != nil {
		return g.errorResponse(c, errorConstant.InvalidQueryParam)
	}
	err = g.validatorInst.Struct(reqData)
	if err != nil {
		return g.errorResponse(c, errorConstant.InvalidQueryParam)
	}

	ctx := c.Context()
	tx := g.db.WithContext(ctx)

	isTemporaryDatacenter := true
	data, err := g.datacenterUC.GetTemporaryDatacenterData(ctx, *reqData.DatacenterID)
	if err != nil {
		isTemporaryDatacenter = false
		data, err = g.datacenterUC.GetDatacenterData(tx, *reqData.DatacenterID)
		if err != nil {
			return g.errorResponse(c, err.Error())
		}
	}
	datacenterData := UCEntity.DatacenterData{
		Credentials: data.Credentials,
		Name:        data.Name,
	}
	googleCredentials, err := g.datacenterUC.GetGoogleCredentials(ctx, datacenterData)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	clusterClient, err := g.clusterUC.GetGoogleClusterClient(ctx, googleCredentials)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	clusters, err := g.clusterUC.GetAllClustersInGCPProject(
		ctx,
		googleCredentials.ProjectID,
		clusterClient,
	)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}

	clusterData := make([]response.GCPCluster, 0)
	for _, cluster := range clusters {
		clusterData = append(
			clusterData, response.GCPCluster{
				Cluster: response.Cluster{
					Name:           cluster.Name,
					Datacenter:     model.GCP,
					DatacenterName: data.Name,
				},
				Location: cluster.Location,
			},
		)
	}

	return g.successResponse(
		c, response.GCPDatacenterClusters{
			Clusters:              clusterData,
			IsTemporaryDatacenter: isTemporaryDatacenter,
		},
	)
}

func (g *gcp) RegisterClusterWithDatacenter(c *fiber.Ctx) error {
	reqData := &request.GCPRegisterClusterData{}
	err := c.BodyParser(reqData)
	if err != nil {
		return g.errorResponse(c, errorConstant.InvalidRequestBody)
	}
	err = g.validatorInst.Struct(reqData)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}

	ctx := c.Context()
	tx := g.db.WithContext(ctx)

	var data *UCEntity.DatacenterDetailedData
	if *reqData.IsDatacenterTemporary {
		data, err = g.datacenterUC.GetTemporaryDatacenterData(ctx, *reqData.DatacenterID)
	} else {
		data, err = g.datacenterUC.GetDatacenterData(tx, *reqData.DatacenterID)
	}
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	datacenterData := UCEntity.DatacenterData{
		Credentials: data.Credentials,
		Name:        data.Name,
	}
	googleCredentials, err := g.datacenterUC.GetGoogleCredentials(ctx, datacenterData)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	clusterClient, err := g.clusterUC.GetGoogleClusterClient(ctx, googleCredentials)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}
	clusters, err := g.clusterUC.GetAllClustersInGCPProject(
		ctx,
		googleCredentials.ProjectID,
		clusterClient,
	)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}

	existingCluster, err := g.generalClusterUC.GetAllClustersInLocalByDatacenterID(
		tx,
		*reqData.DatacenterID,
	)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}

	var selectedClusters []*UCEntity.GCPClusterData
	for _, clusterName := range reqData.ClustersName {
		for _, cluster := range existingCluster {
			if cluster.Name == clusterName {
				return g.errorResponse(c, fmt.Sprintf(errorConstant.ClusterExists, clusterName))
			}
		}

		contains := false
		for _, cluster := range clusters {
			if cluster.Name == clusterName {
				selectedClusters = append(selectedClusters, cluster)
				contains = true
				break
			}
		}
		if !contains {
			return g.errorResponse(c, fmt.Sprintf(errorConstant.ClusterNotFound, clusterName))
		}
	}

	g.clusterUC.RegisterGoogleCredentials(datacenterData.Name, googleCredentials)

	for _, cluster := range clusters {
		kubernetesClient, err := g.clusterUC.GetKubernetesClusterClient(
			datacenterData.Name,
			&cluster.ClusterData,
		)
		if err != nil {
			return g.errorResponse(c, err.Error())
		}
		latestHPAAPIVersion, err := g.generalClusterUC.GetLatestHPAAPIVersion(kubernetesClient)
		if err != nil {
			return g.errorResponse(c, err.Error())
		}
		cluster.LatestHPAAPIVersion = latestHPAAPIVersion
	}

	tx = tx.Begin()

	if *reqData.IsDatacenterTemporary {
		_, err = g.datacenterUC.SaveDatacenterDetailedData(tx, data)
		if err != nil {
			return g.errorResponse(c, err.Error())
		}
	}

	err = g.clusterUC.RegisterClusters(tx, *reqData.DatacenterID, selectedClusters)
	if err != nil {
		return g.errorResponse(c, err.Error())
	}

	tx.Commit()

	responses := make([]response.GCPCluster, 0)
	for _, cluster := range selectedClusters {
		responses = append(
			responses, response.GCPCluster{
				Cluster: response.Cluster{
					ID:             &cluster.ID,
					Name:           cluster.Name,
					Datacenter:     model.GCP,
					DatacenterName: data.Name,
				},
				Location: cluster.Location,
			},
		)
	}

	return g.successResponse(c, responses)
}
