// Package controller provides the worker-status controller.
// Author: magnet2video
// Created: 2026-04-20
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"magnet2video/internal/events/heartbeat"
)

// WorkerController exposes read-only endpoints about worker liveness.
type WorkerController struct {
	statusStore *heartbeat.StatusStore
}

// NewWorkerController constructs a WorkerController. In split deployments the
// statusStore can be nil (unlikely, but the endpoints degrade gracefully).
func NewWorkerController(statusStore *heartbeat.StatusStore) *WorkerController {
	return &WorkerController{statusStore: statusStore}
}

// ListWorkers returns live status for every known worker.
// GET /api/v1/worker/status
func (c *WorkerController) ListWorkers(ctx *gin.Context) {
	if c.statusStore == nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"workers": []any{}, "any_online": false}})
		return
	}
	statuses, err := c.statusStore.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	anyOnline := false
	for _, s := range statuses {
		if s.Online {
			anyOnline = true
			break
		}
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"workers":    statuses,
			"any_online": anyOnline,
		},
	})
}
