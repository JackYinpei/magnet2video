// Package cmd provides application startup and runtime entry point
// Author: Done-0
// Created: 2025-09-25
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/configs"
	"github.com/Done-0/gin-scaffold/internal/cloud"
	cloudHandler "github.com/Done-0/gin-scaffold/internal/cloud/handler"
	cloudTypes "github.com/Done-0/gin-scaffold/internal/cloud/types"
	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/middleware"
	userModel "github.com/Done-0/gin-scaffold/internal/model/user"
	"github.com/Done-0/gin-scaffold/internal/queue"
	"github.com/Done-0/gin-scaffold/internal/transcode/handler"
	"github.com/Done-0/gin-scaffold/internal/transcode/types"
	"github.com/Done-0/gin-scaffold/pkg/router"
	"github.com/Done-0/gin-scaffold/pkg/wire"
	"github.com/Done-0/gin-scaffold/web"
)

// Start starts the HTTP server with graceful shutdown handling
func Start() {
	if err := configs.New(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	cfgs, err := configs.GetConfig()
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	container, err := wire.NewContainer(cfgs)
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	// Initialize managers
	if err := container.LoggerManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer container.LoggerManager.Close()

	if err := container.DatabaseManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer container.DatabaseManager.Close()

	// Create super admin if configured
	if err := createSuperAdmin(cfgs, container.DatabaseManager); err != nil {
		log.Printf("Warning: Failed to create super admin: %v", err)
	}

	if err := container.RedisManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer container.RedisManager.Close()

	// Close torrent manager on shutdown
	defer container.TorrentManager.Close()

	// Close queue producer on shutdown
	defer container.QueueProducer.Close()

	// Start transcode Kafka consumer
	var transcodeConsumer queue.Consumer
	go func() {
		transcodeHandler := handler.NewTranscodeHandler(
			cfgs,
			container.LoggerManager,
			container.DatabaseManager,
			container.QueueProducer,
		)

		var err error
		transcodeConsumer, err = queue.NewConsumer(cfgs, transcodeHandler)
		if err != nil {
			log.Printf("Warning: Failed to create transcode consumer: %v", err)
			return
		}

		if err := transcodeConsumer.Subscribe([]string{types.TopicTranscodeJobs}); err != nil {
			log.Printf("Warning: Failed to subscribe to transcode topic: %v", err)
			return
		}

		log.Println("Transcode Kafka consumer started")
	}()

	// Start cloud upload consumer (if cloud storage is enabled)
	var cloudUploadConsumer queue.Consumer
	if cfgs.CloudStorageConfig.Enabled {
		go func() {
			cloudStorageManager := cloud.New(cfgs, container.LoggerManager)
			defer func() {
				if cloudStorageManager != nil {
					cloudStorageManager.Close()
				}
			}()

			cloudUploadHandler := cloudHandler.NewCloudUploadHandler(
				cfgs,
				container.LoggerManager,
				container.DatabaseManager,
				cloudStorageManager,
			)

			var err error
			cloudUploadConsumer, err = queue.NewConsumer(cfgs, cloudUploadHandler)
			if err != nil {
				log.Printf("Warning: Failed to create cloud upload consumer: %v", err)
				return
			}

			if err := cloudUploadConsumer.Subscribe([]string{cloudTypes.TopicCloudUploadJobs}); err != nil {
				log.Printf("Warning: Failed to subscribe to cloud upload topic: %v", err)
				return
			}

			log.Println("Cloud upload consumer started")
		}()
	}

	// Set Gin mode based on environment
	env := os.Getenv("ENV")
	switch env {
	case "prod", "production":
		gin.SetMode(gin.ReleaseMode)
	case "dev", "development":
		gin.SetMode(gin.DebugMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin engine
	r := gin.New()
	middleware.New(r, cfgs)
	router.New(r, container)

	// Register static file routes for embedded frontend
	web.RegisterStaticRoutes(r)

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%s", cfgs.AppConfig.AppHost, cfgs.AppConfig.AppPort)
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("⇨ Gin server starting on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Close transcode consumer
	if transcodeConsumer != nil {
		transcodeConsumer.Close()
		log.Println("Transcode consumer closed")
	}

	// Close cloud upload consumer
	if cloudUploadConsumer != nil {
		cloudUploadConsumer.Close()
		log.Println("Cloud upload consumer closed")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// createSuperAdmin creates or updates the super admin account based on configuration
func createSuperAdmin(config *configs.Config, dbManager db.DatabaseManager) error {
	email := config.AppConfig.User.SuperAdminEmail
	password := config.AppConfig.User.SuperAdminPassword
	nickname := config.AppConfig.User.SuperAdminNickname

	// Skip if super admin is not configured
	if email == "" || password == "" {
		return nil
	}

	if nickname == "" {
		nickname = "Super Admin"
	}

	var existingAdmin userModel.User
	result := dbManager.DB().Where("email = ?", email).First(&existingAdmin)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new super admin
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		admin := &userModel.User{
			Email:        email,
			Password:     string(hashedPassword),
			Nickname:     nickname,
			Role:         "admin",
			IsSuperAdmin: true,
		}

		if err := dbManager.DB().Create(admin).Error; err != nil {
			return fmt.Errorf("failed to create super admin: %w", err)
		}

		log.Printf("Super admin created: %s", email)
		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("failed to check existing admin: %w", result.Error)
	}

	// Update existing user to be super admin if not already
	if !existingAdmin.IsSuperAdmin || existingAdmin.Role != "admin" {
		if err := dbManager.DB().Model(&existingAdmin).Updates(map[string]any{
			"role":          "admin",
			"is_super_admin": true,
		}).Error; err != nil {
			return fmt.Errorf("failed to update super admin: %w", err)
		}
		log.Printf("User %s upgraded to super admin", email)
	}

	return nil
}
