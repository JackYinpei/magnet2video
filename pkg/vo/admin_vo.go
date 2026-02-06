// Package vo provides admin-related value object definitions
// Author: Done-0
// Created: 2026-01-26
package vo

// AdminUserInfo represents admin view of user information
type AdminUserInfo struct {
	ID           int64  `json:"id"`            // User ID
	Email        string `json:"email"`         // User email
	Nickname     string `json:"nickname"`      // User nickname
	Avatar       string `json:"avatar"`        // User avatar URL
	Role         string `json:"role"`          // User role
	IsSuperAdmin bool   `json:"is_super_admin"` // Whether the user is super admin
	TorrentCount int64  `json:"torrent_count"` // Number of torrents owned
	CreatedAt    int64  `json:"created_at"`    // Account creation timestamp
}

// AdminTorrentInfo represents admin view of torrent information
type AdminTorrentInfo struct {
	ID                int64   `json:"id"`                  // Torrent ID
	InfoHash          string  `json:"info_hash"`           // Info hash
	Name              string  `json:"name"`                // Torrent name
	TotalSize         int64   `json:"total_size"`          // Total size in bytes
	Status            int     `json:"status"`              // Download status
	Progress          float64 `json:"progress"`            // Download progress
	IsPublic          bool    `json:"is_public"`           // Whether the torrent is public (deprecated, use visibility)
	Visibility        int     `json:"visibility"`          // Visibility level: 0=private, 1=internal, 2=public
	TranscodeStatus   int     `json:"transcode_status"`    // Transcode status
	TranscodeProgress int     `json:"transcode_progress"`  // Transcode progress
	CreatorID         int64   `json:"creator_id"`          // Creator user ID
	CreatorNickname   string  `json:"creator_nickname"`    // Creator nickname
	CreatedAt         int64   `json:"created_at"`          // Creation timestamp
}

// ListUsersResponse response for listing users
type ListUsersResponse struct {
	Users    []AdminUserInfo `json:"users"`     // User list
	Total    int64           `json:"total"`     // Total count
	Page     int             `json:"page"`      // Current page
	PageSize int             `json:"page_size"` // Page size
}

// UserDetailResponse response for getting user detail
type UserDetailResponse struct {
	User         AdminUserInfo `json:"user"`          // User info
	TotalStorage int64         `json:"total_storage"` // Total storage used in bytes
}

// UserTorrentsResponse response for getting user torrents
type UserTorrentsResponse struct {
	Torrents []AdminTorrentInfo `json:"torrents"` // Torrent list
	Total    int64              `json:"total"`    // Total count
}

// DeleteUserResponse response for deleting user
type DeleteUserResponse struct {
	Message string `json:"message"` // Status message
}

// UpdateUserRoleResponse response for updating user role
type UpdateUserRoleResponse struct {
	UserID  int64  `json:"user_id"` // User ID
	Role    string `json:"role"`    // New role
	Message string `json:"message"` // Status message
}

// ListAllTorrentsResponse response for listing all torrents
type ListAllTorrentsResponse struct {
	Torrents []AdminTorrentInfo `json:"torrents"`  // Torrent list
	Total    int64              `json:"total"`     // Total count
	Page     int                `json:"page"`      // Current page
	PageSize int                `json:"page_size"` // Page size
}

// DeleteTorrentResponse response for deleting torrent
type DeleteTorrentResponse struct {
	InfoHash string `json:"info_hash"` // Deleted torrent info hash
	Message  string `json:"message"`   // Status message
}

// AdminStatsResponse response for getting system statistics
type AdminStatsResponse struct {
	TotalUsers         int64 `json:"total_users"`          // Total user count
	TotalTorrents      int64 `json:"total_torrents"`       // Total torrent count
	TotalStorage       int64 `json:"total_storage"`        // Total storage in bytes (from database)
	ActualDiskUsage    int64 `json:"actual_disk_usage"`    // Actual disk usage in bytes
	DiskTotal          int64 `json:"disk_total"`           // Total disk capacity in bytes
	DiskFree           int64 `json:"disk_free"`            // Free disk space in bytes
	TranscodingJobs    int64 `json:"transcoding_jobs"`     // Active transcoding jobs
	CompletedDownloads int64 `json:"completed_downloads"`  // Completed downloads count
	ActiveDownloads    int64 `json:"active_downloads"`     // Active downloads count
}

// ResetTranscodeResponse response for resetting transcode
type ResetTranscodeResponse struct {
	InfoHash     string `json:"info_hash"`     // Torrent info hash
	FilesDeleted int    `json:"files_deleted"` // Number of transcoded files deleted
	Message      string `json:"message"`       // Status message
}
