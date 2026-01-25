// Package vo provides user-related value object definitions
// Author: Done-0
// Created: 2026-01-22
package vo

// UserInfo represents basic user information
type UserInfo struct {
	ID       int64  `json:"id"`       // User ID
	Email    string `json:"email"`    // User email
	Nickname string `json:"nickname"` // User nickname
	Avatar   string `json:"avatar"`   // User avatar URL
	Role     string `json:"role"`     // User role
}

// RegisterResponse response for user registration
type RegisterResponse struct {
	User  UserInfo `json:"user"`  // User info
	Token string   `json:"token"` // JWT token
}

// LoginResponse response for user login
type LoginResponse struct {
	User  UserInfo `json:"user"`  // User info
	Token string   `json:"token"` // JWT token
}

// ProfileResponse response for getting user profile
type ProfileResponse struct {
	User UserInfo `json:"user"` // User info
}

// UpdateProfileResponse response for updating user profile
type UpdateProfileResponse struct {
	User    UserInfo `json:"user"`    // Updated user info
	Message string   `json:"message"` // Status message
}

// ChangePasswordResponse response for changing password
type ChangePasswordResponse struct {
	Message string `json:"message"` // Status message
}

// SetTorrentPublicResponse response for setting torrent visibility
type SetTorrentPublicResponse struct {
	InfoHash string `json:"info_hash"` // Torrent info hash
	IsPublic bool   `json:"is_public"` // Whether the torrent is public
	Message  string `json:"message"`   // Status message
}
