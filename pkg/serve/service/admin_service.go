// Package service provides admin service interfaces
// Author: Done-0
// Created: 2026-01-26
package service

import (
	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
	"github.com/Done-0/gin-scaffold/pkg/vo"
)

// AdminService admin service interface
type AdminService interface {
	// ListUsers returns a list of all users
	ListUsers(c *gin.Context, req *dto.ListUsersRequest) (*vo.ListUsersResponse, error)
	// GetUserDetail returns detailed information about a user
	GetUserDetail(c *gin.Context, userID int64) (*vo.UserDetailResponse, error)
	// GetUserTorrents returns all torrents owned by a user
	GetUserTorrents(c *gin.Context, userID int64) (*vo.UserTorrentsResponse, error)
	// DeleteUser deletes a user and optionally their resources
	DeleteUser(c *gin.Context, userID int64) (*vo.DeleteUserResponse, error)
	// UpdateUserRole updates a user's role
	UpdateUserRole(c *gin.Context, req *dto.UpdateUserRoleRequest) (*vo.UpdateUserRoleResponse, error)
	// ListAllTorrents returns a list of all torrents in the system
	ListAllTorrents(c *gin.Context, req *dto.ListAllTorrentsRequest) (*vo.ListAllTorrentsResponse, error)
	// DeleteTorrent deletes a torrent by info hash
	DeleteTorrent(c *gin.Context, infoHash string) (*vo.DeleteTorrentResponse, error)
	// GetStats returns system statistics
	GetStats(c *gin.Context) (*vo.AdminStatsResponse, error)
}
