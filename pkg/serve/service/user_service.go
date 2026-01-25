// Package service provides user service interfaces
// Author: Done-0
// Created: 2026-01-22
package service

import (
	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// UserService user service interface
type UserService interface {
	// Register creates a new user account
	Register(c *gin.Context, req *dto.RegisterRequest) (*vo.RegisterResponse, error)
	// Login authenticates a user and returns a token
	Login(c *gin.Context, req *dto.LoginRequest) (*vo.LoginResponse, error)
	// GetProfile returns the current user's profile
	GetProfile(c *gin.Context) (*vo.ProfileResponse, error)
	// UpdateProfile updates the current user's profile
	UpdateProfile(c *gin.Context, req *dto.UpdateProfileRequest) (*vo.UpdateProfileResponse, error)
	// ChangePassword changes the current user's password
	ChangePassword(c *gin.Context, req *dto.ChangePasswordRequest) (*vo.ChangePasswordResponse, error)
	// SetTorrentPublic sets the visibility of a torrent
	SetTorrentPublic(c *gin.Context, req *dto.SetTorrentPublicRequest) (*vo.SetTorrentPublicResponse, error)
}
