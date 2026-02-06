// Package impl provides user service implementation
// Author: Done-0
// Created: 2026-01-22
package impl

import (
	"errors"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	"github.com/Done-0/gin-scaffold/internal/middleware/auth"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	userModel "github.com/Done-0/gin-scaffold/internal/model/user"
	"github.com/Done-0/gin-scaffold/internal/utils/jwt"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/serve/service"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// UserServiceImpl user service implementation
type UserServiceImpl struct {
	loggerManager logger.LoggerManager
	dbManager     db.DatabaseManager
	jwtConfig     *jwt.JWTConfig
}

// NewUserService creates user service implementation
func NewUserService(
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
) service.UserService {
	return &UserServiceImpl{
		loggerManager: loggerManager,
		dbManager:     dbManager,
		jwtConfig:     jwt.DefaultConfig(),
	}
}

// Register creates a new user account
func (us *UserServiceImpl) Register(c *gin.Context, req *dto.RegisterRequest) (*vo.RegisterResponse, error) {
	// Check if email already exists
	var existingUser userModel.User
	result := us.dbManager.DB().Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		return nil, errors.New("email already registered")
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		us.loggerManager.Logger().Errorf("failed to check existing user: %v", result.Error)
		return nil, result.Error
	}

	// Check if nickname is taken
	result = us.dbManager.DB().Where("nickname = ?", req.Nickname).First(&existingUser)
	if result.Error == nil {
		return nil, errors.New("nickname already taken")
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		us.loggerManager.Logger().Errorf("failed to check existing nickname: %v", result.Error)
		return nil, result.Error
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		us.loggerManager.Logger().Errorf("failed to hash password: %v", err)
		return nil, err
	}

	// Create user
	user := &userModel.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Role:     "user",
	}

	if err := us.dbManager.DB().Create(user).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to create user: %v", err)
		return nil, err
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(us.jwtConfig, user.ID, user.Email, user.Nickname, user.Role)
	if err != nil {
		us.loggerManager.Logger().Errorf("failed to generate token: %v", err)
		return nil, err
	}

	return &vo.RegisterResponse{
		User: vo.UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Role:     user.Role,
		},
		Token: token,
	}, nil
}

// Login authenticates a user and returns a token
func (us *UserServiceImpl) Login(c *gin.Context, req *dto.LoginRequest) (*vo.LoginResponse, error) {
	var user userModel.User
	result := us.dbManager.DB().Where("email = ?", req.Email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("invalid email or password")
	}
	if result.Error != nil {
		us.loggerManager.Logger().Errorf("failed to find user: %v", result.Error)
		return nil, result.Error
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(us.jwtConfig, user.ID, user.Email, user.Nickname, user.Role)
	if err != nil {
		us.loggerManager.Logger().Errorf("failed to generate token: %v", err)
		return nil, err
	}

	return &vo.LoginResponse{
		User: vo.UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Role:     user.Role,
		},
		Token: token,
	}, nil
}

// GetProfile returns the current user's profile
func (us *UserServiceImpl) GetProfile(c *gin.Context) (*vo.ProfileResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}

	var user userModel.User
	if err := us.dbManager.DB().Where("id = ?", userID).First(&user).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to get user profile: %v", err)
		return nil, err
	}

	return &vo.ProfileResponse{
		User: vo.UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Role:     user.Role,
		},
	}, nil
}

// UpdateProfile updates the current user's profile
func (us *UserServiceImpl) UpdateProfile(c *gin.Context, req *dto.UpdateProfileRequest) (*vo.UpdateProfileResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}

	var user userModel.User
	if err := us.dbManager.DB().Where("id = ?", userID).First(&user).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to get user: %v", err)
		return nil, err
	}

	// Check if new nickname is taken by another user
	if req.Nickname != "" && req.Nickname != user.Nickname {
		var existingUser userModel.User
		result := us.dbManager.DB().Where("nickname = ? AND id != ?", req.Nickname, userID).First(&existingUser)
		if result.Error == nil {
			return nil, errors.New("nickname already taken")
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			us.loggerManager.Logger().Errorf("failed to check nickname: %v", result.Error)
			return nil, result.Error
		}
		user.Nickname = req.Nickname
	}

	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := us.dbManager.DB().Save(&user).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to update user: %v", err)
		return nil, err
	}

	return &vo.UpdateProfileResponse{
		User: vo.UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Role:     user.Role,
		},
		Message: "Profile updated successfully",
	}, nil
}

// ChangePassword changes the current user's password
func (us *UserServiceImpl) ChangePassword(c *gin.Context, req *dto.ChangePasswordRequest) (*vo.ChangePasswordResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}

	var user userModel.User
	if err := us.dbManager.DB().Where("id = ?", userID).First(&user).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to get user: %v", err)
		return nil, err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return nil, errors.New("incorrect current password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		us.loggerManager.Logger().Errorf("failed to hash password: %v", err)
		return nil, err
	}

	user.Password = string(hashedPassword)
	if err := us.dbManager.DB().Save(&user).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to update password: %v", err)
		return nil, err
	}

	return &vo.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}

// SetTorrentPublic sets the visibility of a torrent
func (us *UserServiceImpl) SetTorrentPublic(c *gin.Context, req *dto.SetTorrentPublicRequest) (*vo.SetTorrentPublicResponse, error) {
	userID := auth.GetUserID(c)
	if userID == 0 {
		return nil, errors.New("unauthorized")
	}

	if req.Visibility < torrentModel.VisibilityPrivate || req.Visibility > torrentModel.VisibilityPublic {
		return nil, errors.New("invalid visibility value")
	}

	var torrent torrentModel.Torrent
	result := us.dbManager.DB().Where("info_hash = ? AND creator_id = ?", req.InfoHash, userID).First(&torrent)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("torrent not found or not owned by you")
	}
	if result.Error != nil {
		us.loggerManager.Logger().Errorf("failed to find torrent: %v", result.Error)
		return nil, result.Error
	}

	torrent.Visibility = req.Visibility
	if err := us.dbManager.DB().Save(&torrent).Error; err != nil {
		us.loggerManager.Logger().Errorf("failed to update torrent visibility: %v", err)
		return nil, err
	}

	return &vo.SetTorrentPublicResponse{
		InfoHash:   req.InfoHash,
		Visibility: req.Visibility,
		Message:    "Torrent visibility updated successfully",
	}, nil
}
