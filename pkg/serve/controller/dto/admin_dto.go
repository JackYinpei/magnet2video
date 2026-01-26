// Package dto provides admin-related data transfer object definitions
// Author: Done-0
// Created: 2026-01-26
package dto

// ListUsersRequest request for listing users
type ListUsersRequest struct {
	Page     int    `form:"page"`      // Page number (default 1)
	PageSize int    `form:"page_size"` // Page size (default 20)
	Search   string `form:"search"`    // Search by email or nickname
	Role     string `form:"role"`      // Filter by role
}

// UpdateUserRoleRequest request for updating user role
type UpdateUserRoleRequest struct {
	UserID int64  `json:"user_id" validate:"required"` // User ID
	Role   string `json:"role" validate:"required"`    // New role (user or admin)
}

// ListAllTorrentsRequest request for listing all torrents
type ListAllTorrentsRequest struct {
	Page      int    `form:"page"`       // Page number (default 1)
	PageSize  int    `form:"page_size"`  // Page size (default 20)
	Search    string `form:"search"`     // Search by name
	Status    *int   `form:"status"`     // Filter by status
	CreatorID *int64 `form:"creator_id"` // Filter by creator
}
