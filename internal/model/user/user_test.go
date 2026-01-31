// Package user provides user data model tests
// Author: Done-0
// Created: 2026-01-31
package user

import (
	"testing"
)

func TestUser_TableName(t *testing.T) {
	u := User{}
	if u.TableName() != "users" {
		t.Errorf("TableName() = %s, want users", u.TableName())
	}
}

func TestUser_Fields(t *testing.T) {
	u := User{
		Email:        "test@example.com",
		Password:     "hashedpassword",
		Nickname:     "TestUser",
		Avatar:       "https://example.com/avatar.png",
		Role:         "admin",
		IsSuperAdmin: true,
	}

	if u.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", u.Email)
	}
	if u.Password != "hashedpassword" {
		t.Errorf("Password = %s, want hashedpassword", u.Password)
	}
	if u.Nickname != "TestUser" {
		t.Errorf("Nickname = %s, want TestUser", u.Nickname)
	}
	if u.Avatar != "https://example.com/avatar.png" {
		t.Errorf("Avatar = %s, want https://example.com/avatar.png", u.Avatar)
	}
	if u.Role != "admin" {
		t.Errorf("Role = %s, want admin", u.Role)
	}
	if !u.IsSuperAdmin {
		t.Error("IsSuperAdmin should be true")
	}
}

func TestUser_DefaultValues(t *testing.T) {
	u := User{}

	// Default values should be zero values
	if u.Email != "" {
		t.Errorf("Default Email should be empty, got %s", u.Email)
	}
	if u.IsSuperAdmin {
		t.Error("Default IsSuperAdmin should be false")
	}
}

func TestUser_EmbeddedBase(t *testing.T) {
	u := User{}

	// User should have access to Base fields
	if u.ID != 0 {
		t.Errorf("Initial ID should be 0, got %d", u.ID)
	}
	if u.CreatedAt != 0 {
		t.Errorf("Initial CreatedAt should be 0, got %d", u.CreatedAt)
	}
	if u.UpdatedAt != 0 {
		t.Errorf("Initial UpdatedAt should be 0, got %d", u.UpdatedAt)
	}
	if u.Deleted {
		t.Error("Initial Deleted should be false")
	}
}

func TestUser_Roles(t *testing.T) {
	tests := []struct {
		role     string
		expected string
	}{
		{"user", "user"},
		{"admin", "admin"},
		{"moderator", "moderator"},
	}

	for _, tt := range tests {
		u := User{Role: tt.role}
		if u.Role != tt.expected {
			t.Errorf("Role = %s, want %s", u.Role, tt.expected)
		}
	}
}
