// Package impl provides admin service implementation
// Author: Done-0
// Created: 2026-01-26
package impl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"magnet2video/internal/db"
	"magnet2video/internal/events/heartbeat"
	eventTypes "magnet2video/internal/events/types"
	"magnet2video/internal/logger"
	"magnet2video/internal/middleware/auth"
	torrentModel "magnet2video/internal/model/torrent"
	transcodeModel "magnet2video/internal/model/transcode"
	userModel "magnet2video/internal/model/user"
	"magnet2video/internal/queue"
	torrentTypes "magnet2video/internal/torrent/types"
	"magnet2video/pkg/serve/controller/dto"
	"magnet2video/pkg/serve/service"
	"magnet2video/pkg/vo"
)

// AdminServiceImpl admin service implementation.
//
// AdminServiceImpl never touches the worker's filesystem or torrent client
// directly. Removals are dispatched as queue messages (download-jobs for
// torrent state, file-ops-jobs for derived files), and disk stats come from
// worker heartbeats via the status store.
type AdminServiceImpl struct {
	loggerManager logger.LoggerManager
	dbManager     db.DatabaseManager
	queueProducer queue.Producer
	statusStore   *heartbeat.StatusStore
}

// NewAdminService creates admin service implementation
func NewAdminService(
	loggerManager logger.LoggerManager,
	dbManager db.DatabaseManager,
	queueProducer queue.Producer,
	statusStore *heartbeat.StatusStore,
) service.AdminService {
	return &AdminServiceImpl{
		loggerManager: loggerManager,
		dbManager:     dbManager,
		queueProducer: queueProducer,
		statusStore:   statusStore,
	}
}

// publishDownloadJob sends a download control command to the worker.
func (as *AdminServiceImpl) publishDownloadJob(ctx context.Context, job eventTypes.DownloadJob) {
	if as.queueProducer == nil {
		as.loggerManager.Logger().Warn("queue producer unavailable; download command dropped")
		return
	}
	data, err := json.Marshal(job)
	if err != nil {
		as.loggerManager.Logger().Errorf("marshal download job: %v", err)
		return
	}
	if err := as.queueProducer.Send(ctx, eventTypes.TopicDownloadJobs, nil, data); err != nil {
		as.loggerManager.Logger().Errorf("publish download job failed: action=%s infoHash=%s err=%v",
			job.Action, job.InfoHash, err)
	}
}

// publishFileOp sends a filesystem-mutating command to the worker.
func (as *AdminServiceImpl) publishFileOp(ctx context.Context, op torrentTypes.FileOpMessage) {
	if as.queueProducer == nil {
		as.loggerManager.Logger().Warn("queue producer unavailable; file op dropped")
		return
	}
	data, err := json.Marshal(op)
	if err != nil {
		as.loggerManager.Logger().Errorf("marshal file op: %v", err)
		return
	}
	if err := as.queueProducer.Send(ctx, torrentTypes.TopicFileOps, nil, data); err != nil {
		as.loggerManager.Logger().Errorf("publish file op failed: op=%s torrent=%d err=%v",
			op.Op, op.TorrentID, err)
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
			IsPublic:          t.Visibility >= torrentModel.VisibilityPublic,
			Visibility:        t.Visibility,
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

	// Find user's torrents and dispatch removal commands to the worker.
	// We do NOT touch the worker's filesystem or torrent client from here;
	// the worker handles RemoveTorrent (action=remove + DeleteFiles=true)
	// which both untracks the torrent and deletes its on-disk files.
	var torrents []torrentModel.Torrent
	as.dbManager.DB().Where("creator_id = ?", userID).Find(&torrents)
	ctx := c.Request.Context()
	for _, t := range torrents {
		as.publishDownloadJob(ctx, eventTypes.DownloadJob{
			Action:      eventTypes.DownloadActionRemove,
			InfoHash:    t.InfoHash,
			DeleteFiles: true,
		})
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
			IsPublic:          t.Visibility >= torrentModel.VisibilityPublic,
			Visibility:        t.Visibility,
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

	// Tell the worker to stop seeding/downloading and delete the on-disk
	// files. The worker is the only side that owns the disk.
	as.publishDownloadJob(c.Request.Context(), eventTypes.DownloadJob{
		Action:      eventTypes.DownloadActionRemove,
		InfoHash:    infoHash,
		DeleteFiles: true,
	})

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

// ResetTranscode resets transcode status and deletes transcoded files
func (as *AdminServiceImpl) ResetTranscode(c *gin.Context, infoHash string) (*vo.ResetTranscodeResponse, error) {
	var torrentRecord torrentModel.Torrent
	if err := as.dbManager.DB().Where("info_hash = ?", infoHash).First(&torrentRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("torrent not found")
		}
		as.loggerManager.Logger().Errorf("failed to get torrent: %v", err)
		return nil, err
	}

	// Load all files from torrent_files table
	var allFiles []torrentModel.TorrentFile
	if err := as.dbManager.DB().Where("torrent_id = ?", torrentRecord.ID).Find(&allFiles).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to load torrent files: %v", err)
		return nil, err
	}

	// Collect derived file paths and DB ids. Physical removal is delegated
	// to the worker via file-ops-jobs — the server has no business doing
	// os.Remove on a path it doesn't own.
	var derivedPaths []string
	var derivedIDs []int64
	for _, f := range allFiles {
		if f.Source == "transcoded" || f.Source == "extracted" {
			if f.Path != "" {
				derivedPaths = append(derivedPaths, f.Path)
			}
			derivedIDs = append(derivedIDs, f.ID)
		}
	}

	if len(derivedPaths) > 0 {
		as.publishFileOp(c.Request.Context(), torrentTypes.FileOpMessage{
			Op:           torrentTypes.FileOpDeleteDerived,
			OpID:         eventTypes.GenerateEventID(),
			TorrentID:    torrentRecord.ID,
			InfoHash:     infoHash,
			DownloadPath: torrentRecord.DownloadPath,
			TorrentName:  torrentRecord.Name,
			Paths:        derivedPaths,
		})
	}

	// Delete derived file records from torrent_files
	if len(derivedIDs) > 0 {
		if err := as.dbManager.DB().Where("id IN ?", derivedIDs).Delete(&torrentModel.TorrentFile{}).Error; err != nil {
			as.loggerManager.Logger().Errorf("failed to delete derived file records: %v", err)
			return nil, err
		}
	}

	// Reset transcode status on original files
	if err := as.dbManager.DB().Model(&torrentModel.TorrentFile{}).
		Where("torrent_id = ? AND (source = '' OR source = 'original')", torrentRecord.ID).
		Updates(map[string]any{
			"transcode_status": torrentModel.TranscodeStatusNone,
			"transcoded_path":  "",
			"transcode_error":  "",
		}).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to reset file transcode status: %v", err)
		return nil, err
	}

	// Reset torrent-level transcode status
	if err := as.dbManager.DB().Model(&torrentRecord).Updates(map[string]any{
		"transcode_status":   torrentModel.TranscodeStatusNone,
		"transcode_progress": 0,
		"transcoded_count":   0,
		"total_transcode":    0,
	}).Error; err != nil {
		as.loggerManager.Logger().Errorf("failed to reset transcode status: %v", err)
		return nil, err
	}

	// Delete related transcode jobs
	as.dbManager.DB().Where("info_hash = ?", infoHash).Delete(&transcodeModel.TranscodeJob{})

	currentUserID := auth.GetUserID(c)
	as.loggerManager.Logger().Infof("Transcode reset by admin: infoHash=%s, derivedDispatched=%d, resetBy=%d",
		infoHash, len(derivedPaths), currentUserID)

	return &vo.ResetTranscodeResponse{
		InfoHash:     infoHash,
		FilesDeleted: len(derivedPaths),
		Message:      fmt.Sprintf("Transcode reset; %d derived file(s) dispatched for worker deletion", len(derivedPaths)),
	}, nil
}

// GetStats returns system statistics. Disk numbers come from worker
// heartbeats — the server side has no idea where the actual storage is in
// split deployment, so we read them from whoever last reported.
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

	const gb = int64(1024 * 1024 * 1024)
	var diskTotal, diskFree int64
	if as.statusStore != nil {
		summary := as.statusStore.AggregateDisk(c.Request.Context())
		diskTotal = summary.TotalGB * gb
		diskFree = summary.FreeGB * gb
	}

	return &vo.AdminStatsResponse{
		TotalUsers:         totalUsers,
		TotalTorrents:      totalTorrents,
		TotalStorage:       totalStorage,
		ActualDiskUsage:    totalStorage, // best-effort: derived from DB sum
		DiskTotal:          diskTotal,
		DiskFree:           diskFree,
		TranscodingJobs:    transcodingJobs,
		CompletedDownloads: completedDownloads,
		ActiveDownloads:    activeDownloads,
	}, nil
}
