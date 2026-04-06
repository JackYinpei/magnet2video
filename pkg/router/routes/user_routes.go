// Package routes provides user route registration functionality
// Author: Done-0
// Created: 2026-01-22
package routes

import (
	"github.com/gin-gonic/gin"

	"magnet2video/internal/middleware/auth"
	"magnet2video/pkg/wire"
)

// RegisterUserRoutes registers user module routes
func RegisterUserRoutes(container *wire.Container, v1, v2 *gin.RouterGroup) {
	// V1 routes - Authentication (no auth required)
	authGroup := v1.Group("/auth")
	{
		// Register new user
		authGroup.POST("/register", container.UserController.Register)

		// Login
		authGroup.POST("/login", container.UserController.Login)
	}

	// V1 routes - User profile (auth required)
	userGroup := v1.Group("/user")
	userGroup.Use(auth.JWTMiddleware())
	{
		// Get current user profile
		userGroup.GET("/profile", container.UserController.GetProfile)

		// Update profile
		userGroup.PUT("/profile", container.UserController.UpdateProfile)

		// Change password
		userGroup.PUT("/password", container.UserController.ChangePassword)

		// Set torrent visibility
		userGroup.POST("/torrent/public", container.UserController.SetTorrentPublic)
	}
}
