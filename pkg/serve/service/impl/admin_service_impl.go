// Package impl provides admin service implementation
// Author: Done-0
// Created: 2026-01-26
package impl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Done-0/gin-scaffold/internal/db"
	"github.com/Done-0/gin-scaffold/internal/logger"
	"github.com/Done-0/gin-scaffold/internal/middleware/auth"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
	userModel "github.com/Done-0/gin-scaffold/internal/model/user"
	"github.com/Done-0/gin-scaffold/internal/torrent"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/serve/service"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// AdminServiceImpl admin service implementation
type AdminServiceImpl struct {
	loggerManager  logger.LoggerManager
	dbManager      db.DatabaseManager
	torrentManager torrent.TorrentManager
}

// NewAdminService creates admin service implementation
func NewAdminService(
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	torrentManager torrent.TorrentManager,
) service.AdminService {
	return &AdminServiceImpl{
		loggerManager:  loggerManager,
		dbManager:      dbManager,
		torrentManager: torrentManager,
	}
}

// ListUsers returns a list of all users
func (as *AdminServiceImpl) ListUsers(c *gin.Context, req *dto.ListUsersRequest) (*vo.ListUsersResponse, error) {
	var users []userModel.User
	query := as.dbManager.DB().Model(&userModel.User{})

	// Apply filters
	if req.Search != "" {
		query = query.Where("email LIKE ? OR nickname LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}
	if req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to count users: %v", err)
		return nil, err
	}

	// Pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to list users: %v", err)
		return nil, err
	}

	// Build response
	userList := make([]vo.AdminUserInfo, 0, len(users))
	for _, user := range users {
		// Count user's torrents
		var torrentCount int64
		as.dbManager.DB().Model(&torrentModel.Torrent{}).Where("creator_id = ?", user.ID).Count(&torrentCount)

		userList = append(userList, vo.AdminUserInfo{
			ID:           user.ID,
			Email:        user.Email,
			Nickname:     user.Nickname,
			Avatar:       user.Avatar,
			Role:         user.Role,
			IsSuperAdmin: user.IsSuperAdmin,
			TorrentCount: torrentCount,
			CreatedAt:    user.CreatedAt,
		})
	}

	return &vo.ListUsersResponse{
		Users:    userList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUserDetail returns detailed information about a user
func (as *AdminServiceImpl) GetUserDetail(c *gin.Context, userID int64) (*vo.UserDetailResponse, error) {
	var user userModel.User
	if err := as.dbManager.DB().Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		as.loggerManager.Logger().Errorf("failed to get user: %v", err)
		return nil, err
	}

	// Count user's torrents
	var torrentCount int64
	as.dbManager.DB().Model(&torrentModel.Torrent{}).Where("creator_id = ?", user.ID).Count(&torrentCount)

	// Calculate total storage used
	var totalStorage int64
	as.dbManager.DB().Model(&torrentModel.Torrent{}).Where("creator_id = ?", user.ID).Select("COALESCE(SUM(total_size), 0)").Scan(&totalStorage)

	return &vo.UserDetailResponse{
		User: vo.AdminUserInfo{
			ID:           user.ID,
			Email:        user.Email,
			Nickname:     user.Nickname,
			Avatar:       user.Avatar,
			Role:         user.Role,
			IsSuperAdmin: user.IsSuperAdmin,
			TorrentCount: torrentCount,
			CreatedAt:    user.CreatedAt,
		},
		TotalStorage: totalStorage,
	}, nil
}

// GetUserTorrents returns all torrents owned by a user
func (as *AdminServiceImpl) GetUserTorrents(c *gin.Context, userID int64) (*vo.UserTorrentsResponse, error) {
	var torrents []torrentModel.Torrent
	if err := as.dbManager.DB().Where("creator_id = ?", userID).Order("created_at DESC").Find(&torrents).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to get user torrents: %v", err)
		return nil, err
	}

	torrentList := make([]vo.AdminTorrentInfo, 0, len(torrents))
	for _, t := range torrents {
		torrentList = append(torrentList, vo.AdminTorrentInfo{
			ID:                t.ID,
			InfoHash:          t.InfoHash,
			Name:              t.Name,
			TotalSize:         t.TotalSize,
			Status:            t.Status,
			Progress:          t.Progress,
			IsPublic:          t.IsPublic,
			TranscodeStatus:   t.TranscodeStatus,
			TranscodeProgress: t.TranscodeProgress,
			CreatorID:         t.CreatorID,
			CreatedAt:         t.CreatedAt,
		})
	}

	return &vo.UserTorrentsResponse{
		Torrents: torrentList,
		Total:    int64(len(torrentList)),
	}, nil
}

// DeleteUser deletes a user and optionally their resources
func (as *AdminServiceImpl) DeleteUser(c *gin.Context, userID int64) (*vo.DeleteUserResponse, error) {
	currentUserID := auth.GetUserID(c)
	if currentUserID == userID {
		return nil, errors.New("cannot delete your own account")
	}

	var user userModel.User
	if err := as.dbManager.DB().Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		as.loggerManager.Logger().Errorf("failed to get user: %v", err)
		return nil, err
	}

	// Check if user is super admin
	if user.IsSuperAdmin {
		return nil, errors.New("cannot delete super admin account")
	}

	// Delete user's torrents first
	var torrents []torrentModel.Torrent
	as.dbManager.DB().Where("creator_id = ?", userID).Find(&torrents)

	for _, t := range torrents {
		// Remove from torrent client
		as.torrentManager.Client().RemoveTorrent(t.InfoHash, true)

		// Delete download files
		if t.DownloadPath != "" {
			os.RemoveAll(t.DownloadPath)
		}
	}

	// Delete torrents from database
	if err := as.dbManager.DB().Where("creator_id = ?", userID).Delete(&torrentModel.Torrent{}).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to delete user torrents: %v", err)
		return nil, err
	}

	// Delete transcode jobs
	if err := as.dbManager.DB().Where("creator_id = ?", userID).Delete(&transcodeModel.TranscodeJob{}).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to delete user transcode jobs: %v", err)
	}

	// Delete user
	if err := as.dbManager.DB().Delete(&user).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to delete user: %v", err)
		return nil, err
	}

	as.loggerManager.Logger().Infof("User deleted by admin: userID=%d, deletedBy=%d", userID, currentUserID)

	return &vo.DeleteUserResponse{
		Message: fmt.Sprintf("User %s deleted successfully", user.Email),
	}, nil
}

// UpdateUserRole updates a user's role
func (as *AdminServiceImpl) UpdateUserRole(c *gin.Context, req *dto.UpdateUserRoleRequest) (*vo.UpdateUserRoleResponse, error) {
	currentUserID := auth.GetUserID(c)
	if currentUserID == req.UserID {
		return nil, errors.New("cannot modify your own role")
	}

	var user userModel.User
	if err := as.dbManager.DB().Where("id = ?", req.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		as.loggerManager.Logger().Errorf("failed to get user: %v", err)
		return nil, err
	}

	// Validate role
	if req.Role != "user" && req.Role != "admin" {
		return nil, errors.New("invalid role, must be 'user' or 'admin'")
	}

	user.Role = req.Role
	if err := as.dbManager.DB().Save(&user).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to update user role: %v", err)
		return nil, err
	}

	as.loggerManager.Logger().Infof("User role updated: userID=%d, newRole=%s, updatedBy=%d", req.UserID, req.Role, currentUserID)

	return &vo.UpdateUserRoleResponse{
		UserID:  req.UserID,
		Role:    req.Role,
		Message: "User role updated successfully",
	}, nil
}

// ListAllTorrents returns a list of all torrents in the system
func (as *AdminServiceImpl) ListAllTorrents(c *gin.Context, req *dto.ListAllTorrentsRequest) (*vo.ListAllTorrentsResponse, error) {
	var torrents []torrentModel.Torrent
	query := as.dbManager.DB().Model(&torrentModel.Torrent{})

	// Apply filters
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.CreatorID != nil {
		query = query.Where("creator_id = ?", *req.CreatorID)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to count torrents: %v", err)
		return nil, err
	}

	// Pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&torrents).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to list torrents: %v", err)
		return nil, err
	}

	// Build response with creator info
	torrentList := make([]vo.AdminTorrentInfo, 0, len(torrents))
	for _, t := range torrents {
		info := vo.AdminTorrentInfo{
			ID:                t.ID,
			InfoHash:          t.InfoHash,
			Name:              t.Name,
			TotalSize:         t.TotalSize,
			Status:            t.Status,
			Progress:          t.Progress,
			IsPublic:          t.IsPublic,
			TranscodeStatus:   t.TranscodeStatus,
			TranscodeProgress: t.TranscodeProgress,
			CreatorID:         t.CreatorID,
			CreatedAt:         t.CreatedAt,
		}

		// Get creator nickname
		if t.CreatorID > 0 {
			var creator userModel.User
			if as.dbManager.DB().Select("nickname").Where("id = ?", t.CreatorID).First(&creator).Error == nil {
				info.CreatorNickname = creator.Nickname
			}
		}

		torrentList = append(torrentList, info)
	}

	return &vo.ListAllTorrentsResponse{
		Torrents: torrentList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// DeleteTorrent deletes a torrent by info hash
func (as *AdminServiceImpl) DeleteTorrent(c *gin.Context, infoHash string) (*vo.DeleteTorrentResponse, error) {
	var torrentRecord torrentModel.Torrent
	if err := as.dbManager.DB().Where("info_hash = ?", infoHash).First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("torrent not found")
		}
		as.loggerManager.Logger().Errorf("failed to get torrent: %v", err)
		return nil, err
	}

	// Remove from torrent client
	as.torrentManager.Client().RemoveTorrent(infoHash, true)

	// Delete download files
	if torrentRecord.DownloadPath != "" {
		if err := os.RemoveAll(torrentRecord.DownloadPath); err != nil {
			as.loggerManager.Logger().Warnf("failed to remove download files: %v", err)
		}
	}

	// Delete related transcode jobs
	as.dbManager.DB().Where("info_hash = ?", infoHash).Delete(&transcodeModel.TranscodeJob{})

	// Delete torrent record
	if err := as.dbManager.DB().Delete(&torrentRecord).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to delete torrent: %v", err)
		return nil, err
	}

	currentUserID := auth.GetUserID(c)
	as.loggerManager.Logger().Infof("Torrent deleted by admin: infoHash=%s, deletedBy=%d", infoHash, currentUserID)

	return &vo.DeleteTorrentResponse{
		InfoHash: infoHash,
		Message:  "Torrent deleted successfully",
	}, nil
}

// GetStats returns system statistics
func (as *AdminServiceImpl) GetStats(c *gin.Context) (*vo.AdminStatsResponse, error) {
	var totalUsers int64
	as.dbManager.DB().Model(&userModel.User{}).Count(&totalUsers)

	var totalTorrents int64
	as.dbManager.DB().Model(&torrentModel.Torrent{}).Count(&totalTorrents)

	var totalStorage int64
	as.dbManager.DB().Model(&torrentModel.Torrent{}).Select("COALESCE(SUM(total_size), 0)").Scan(&totalStorage)

	var transcodingJobs int64
	as.dbManager.DB().Model(&transcodeModel.TranscodeJob{}).Where("status IN (?, ?)", transcodeModel.JobStatusPending, transcodeModel.JobStatusProcessing).Count(&transcodingJobs)

	var completedDownloads int64
	as.dbManager.DB().Model(&torrentModel.Torrent{}).Where("status = ?", torrentModel.StatusCompleted).Count(&completedDownloads)

	var activeDownloads int64
	as.dbManager.DB().Model(&torrentModel.Torrent{}).Where("status = ?", torrentModel.StatusDownloading).Count(&activeDownloads)

	// Calculate actual disk usage
	var actualDiskUsage int64
	downloadDir := as.torrentManager.Client().GetDownloadDir()
	if downloadDir != "" {
		filepath.Walk(downloadDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				actualDiskUsage += info.Size()
			}
			return nil
		})
	}

	return &vo.AdminStatsResponse{
		TotalUsers:         totalUsers,
		TotalTorrents:      totalTorrents,
		TotalStorage:       totalStorage,
		ActualDiskUsage:    actualDiskUsage,
		TranscodingJobs:    transcodingJobs,
		CompletedDownloads: completedDownloads,
		ActiveDownloads:    activeDownloads,
	}, nil
}
