// Package jwt provides JWT token generation and validation tests
// Author: Done-0
// Created: 2026-01-31
package jwt

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.SecretKey == "" {
		t.Error("DefaultConfig() SecretKey should not be empty")
	}
	if config.TokenDuration <= 0 {
		t.Error("DefaultConfig() TokenDuration should be positive")
	}
	if config.Issuer == "" {
		t.Error("DefaultConfig() Issuer should not be empty")
	}
}

func TestGenerateToken(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		name     string
		userID   int64
		email    string
		nickname string
		role     string
		wantErr  bool
	}{
		{
			name:     "valid user",
			userID:   12345,
			email:    "test@example.com",
			nickname: "TestUser",
			role:     "user",
			wantErr:  false,
		},
		{
			name:     "admin user",
			userID:   1,
			email:    "admin@example.com",
			nickname: "Admin",
			role:     "admin",
			wantErr:  false,
		},
		{
			name:     "empty fields",
			userID:   0,
			email:    "",
			nickname: "",
			role:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(config, tt.userID, tt.email, tt.nickname, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("GenerateToken() returned empty token")
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	config := DefaultConfig()

	// Generate a valid token first
	userID := int64(12345)
	email := "test@example.com"
	nickname := "TestUser"
	role := "user"

	token, err := GenerateToken(config, userID, email, nickname, role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Parse the token
	claims, err := ParseToken(config, token)
	if err != nil {
		t.Errorf("ParseToken() error = %v", err)
		return
	}

	if claims.UserID != userID {
		t.Errorf("ParseToken() UserID = %d, want %d", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("ParseToken() Email = %s, want %s", claims.Email, email)
	}
	if claims.Nickname != nickname {
		t.Errorf("ParseToken() Nickname = %s, want %s", claims.Nickname, nickname)
	}
	if claims.Role != role {
		t.Errorf("ParseToken() Role = %s, want %s", claims.Role, role)
	}
}

func TestParseToken_InvalidToken(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "invalid.token.format",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "random string",
			token:   "randomstring",
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToken(config, tt.token)
			if err == nil {
				t.Error("ParseToken() should return error for invalid token")
			}
		})
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	config1 := &JWTConfig{
		SecretKey:     "secret1",
		TokenDuration: 24 * time.Hour,
		Issuer:        "test",
	}
	config2 := &JWTConfig{
		SecretKey:     "secret2",
		TokenDuration: 24 * time.Hour,
		Issuer:        "test",
	}

	token, err := GenerateToken(config1, 123, "test@test.com", "Test", "user")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to parse with different secret
	_, err = ParseToken(config2, token)
	if err != ErrInvalidToken {
		t.Errorf("ParseToken() with wrong secret should return ErrInvalidToken, got %v", err)
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	config := &JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: -1 * time.Hour, // Already expired
		Issuer:        "test",
	}

	token, err := GenerateToken(config, 123, "test@test.com", "Test", "user")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	_, err = ParseToken(config, token)
	if err != ErrExpiredToken {
		t.Errorf("ParseToken() with expired token should return ErrExpiredToken, got %v", err)
	}
}

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	config := DefaultConfig()

	testCases := []struct {
		userID   int64
		email    string
		nickname string
		role     string
	}{
		{1, "user1@test.com", "User1", "user"},
		{999999999, "admin@test.com", "SuperAdmin", "admin"},
		{42, "special@test.com", "特殊用户", "moderator"},
	}

	for _, tc := range testCases {
		token, err := GenerateToken(config, tc.userID, tc.email, tc.nickname, tc.role)
		if err != nil {
			t.Errorf("GenerateToken() error = %v", err)
			continue
		}

		claims, err := ParseToken(config, token)
		if err != nil {
			t.Errorf("ParseToken() error = %v", err)
			continue
		}

		if claims.UserID != tc.userID ||
			claims.Email != tc.email ||
			claims.Nickname != tc.nickname ||
			claims.Role != tc.role {
			t.Errorf("Round-trip failed: got %+v, want userID=%d email=%s nickname=%s role=%s",
				claims, tc.userID, tc.email, tc.nickname, tc.role)
		}
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	config := DefaultConfig()
	for i := 0; i < b.N; i++ {
		_, err := GenerateToken(config, 12345, "test@example.com", "TestUser", "user")
		if err != nil {
			b.Fatalf("GenerateToken() error = %v", err)
		}
	}
}

func BenchmarkParseToken(b *testing.B) {
	config := DefaultConfig()
	token, _ := GenerateToken(config, 12345, "test@example.com", "TestUser", "user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseToken(config, token)
		if err != nil {
			b.Fatalf("ParseToken() error = %v", err)
		}
	}
}
