// Package dto provides user-related data transfer object definitions
// Author: Done-0
// Created: 2026-01-22
package dto

// RegisterRequest request for user registration
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`    // User email
	Password string `json:"password" validate:"required,min=6"` // User password, min 6 characters
	Nickname string `json:"nickname" validate:"required,min=2"` // User nickname, min 2 characters
}

// LoginRequest request for user login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`    // User email
	Password string `json:"password" validate:"required,min=6"` // User password
}

// UpdateProfileRequest request for updating user profile
type UpdateProfileRequest struct {
	Nickname string `json:"nickname"` // User nickname
	Avatar   string `json:"avatar"`   // User avatar URL
}

// ChangePasswordRequest request for changing password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`       // Current password
	NewPassword string `json:"new_password" validate:"required,min=6"` // New password
}

// SetTorrentPublicRequest request for setting torrent visibility
type SetTorrentPublicRequest struct {
	InfoHash string `json:"info_hash" validate:"required"` // Torrent info hash
	IsPublic bool   `json:"is_public"`                     // Whether to make the torrent public
}
