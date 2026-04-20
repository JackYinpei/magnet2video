// Package cmd: shared helpers used by every mode bootstrap.
// Author: magnet2video
// Created: 2026-04-20
package cmd

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"magnet2video/configs"
	"magnet2video/internal/db"
	userModel "magnet2video/internal/model/user"
)

// createSuperAdmin creates or updates the super admin account based on configuration.
func createSuperAdmin(config *configs.Config, dbManager db.DatabaseManager) error {
	email := config.AppConfig.User.SuperAdminEmail
	password := config.AppConfig.User.SuperAdminPassword
	nickname := config.AppConfig.User.SuperAdminNickname

	if email == "" || password == "" {
		return nil
	}
	if nickname == "" {
		nickname = "Super Admin"
	}

	var existingAdmin userModel.User
	result := dbManager.DB().Where("email = ?", email).First(&existingAdmin)

	if result.Error == gorm.ErrRecordNotFound {
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

	if !existingAdmin.IsSuperAdmin || existingAdmin.Role != "admin" {
		if err := dbManager.DB().Model(&existingAdmin).Updates(map[string]any{
			"role":           "admin",
			"is_super_admin": true,
		}).Error; err != nil {
			return fmt.Errorf("failed to update super admin: %w", err)
		}
		log.Printf("User %s upgraded to super admin", email)
	}
	return nil
}
