// Package routes provides admin route registration functionality
// Author: Done-0
// Created: 2026-01-26
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/internal/middleware/auth"
	"github.com/Done-0/gin-scaffold/pkg/wire"
)

// RegisterAdminRoutes registers admin module routes
func RegisterAdminRoutes(container *wire.Container, v1, v2 *gin.RouterGroup) {
	// V1 routes - Admin management (requires JWT + Admin role)
	adminGroup := v1.Group("/admin")
	adminGroup.Use(auth.JWTMiddleware())
	adminGroup.Use(auth.AdminMiddleware())
	{
		// User management
		adminGroup.GET("/users", container.AdminController.ListUsers)
		adminGroup.GET("/users/:id", container.AdminController.GetUserDetail)
		adminGroup.GET("/users/:id/torrents", container.AdminController.GetUserTorrents)
		adminGroup.DELETE("/users/:id", container.AdminController.DeleteUser)
		adminGroup.PUT("/users/:id/role", container.AdminController.UpdateUserRole)

		// Torrent management
		adminGroup.GET("/torrents", container.AdminController.ListAllTorrents)
		adminGroup.DELETE("/torrents/:info_hash", container.AdminController.DeleteTorrent)

		// System statistics
		adminGroup.GET("/stats", container.AdminController.GetStats)
	}
}
