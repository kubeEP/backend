package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/handler"
)

func buildRoute(handlers *handler.Handlers, router fiber.Router) {
	router.Use(
		cors.New(
			cors.Config{
				AllowHeaders: "Origin, Content-Type, Accept",
				AllowOrigins: "http://localhost:3000",
			},
		),
	)

	router.Route(
		"/gcp", func(router fiber.Router) {
			router.Route(
				"/register", func(router fiber.Router) {
					router.Post("/datacenter", handlers.GcpHandler.RegisterDatacenter)
					router.Post("/clusters", handlers.GcpHandler.RegisterClusterWithDatacenter)
				},
			)
			router.Get("/clusters", handlers.GcpHandler.GetClustersByDatacenterID)
		},
	)

	router.Route(
		"/cluster", func(router fiber.Router) {
			router.Get("/list", handlers.ClusterHandler.GetAllRegisteredClusters)
			router.Route(
				"/:cluster_id", func(router fiber.Router) {
					router.Get("/hpa", handlers.ClusterHandler.GetClusterAllHPA)
					router.Get("/", handlers.ClusterHandler.GetClusterSimpleData)
				},
			)
		},
	)

	router.Route(
		"/event", func(router fiber.Router) {
			router.Post("/register", handlers.EventHandler.RegisterEvents)
			router.Put("/update", handlers.EventHandler.UpdateEvent)
			router.Get("/list", handlers.EventHandler.ListEventByCluster)
			router.Get(
				"/status/node-pool/:updated_node_pool_id",
				handlers.EventHandler.ListNodePoolStatusByUpdatedNodePool,
			)
			router.Get(
				"/status/hpa/:scheduled_hpa_config_id",
				handlers.EventHandler.ListHPAStatusByScheduledHPAConfig,
			)
			router.Get("/:event_id", handlers.EventHandler.GetDetailedEvent)
			router.Delete("/:event_id", handlers.EventHandler.DeleteEvent)
		},
	)
}
