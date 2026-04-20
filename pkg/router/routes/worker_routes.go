// Package routes: registers worker-status read-only endpoints.
// Author: magnet2video
// Created: 2026-04-20
package routes

import (
	"github.com/gin-gonic/gin"

	"magnet2video/pkg/wire"
)

// RegisterWorkerRoutes registers worker-status routes.
// These endpoints are public (read-only) so the frontend can show a status
// banner without requiring authentication.
func RegisterWorkerRoutes(container *wire.Container, v1 *gin.RouterGroup) {
	if container.WorkerController == nil {
		return
	}
	workerGroup := v1.Group("/worker")
	{
		workerGroup.GET("/status", container.WorkerController.ListWorkers)
	}
}
