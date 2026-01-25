// Package auth provides JWT authentication middleware
// Author: Done-0
// Created: 2026-01-22
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Done-0/gin-scaffold/internal/types/errno"
	"github.com/Done-0/gin-scaffold/internal/utils/errorx"
	"github.com/Done-0/gin-scaffold/internal/utils/jwt"
	"github.com/Done-0/gin-scaffold/internal/utils/vo"
)

const (
	// AuthorizationHeader is the header key for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer token
	BearerPrefix = "Bearer "
	// UserIDKey is the context key for user ID
	UserIDKey = "user_id"
	// UserEmailKey is the context key for user email
	UserEmailKey = "user_email"
	// UserNicknameKey is the context key for user nickname
	UserNicknameKey = "user_nickname"
	// UserRoleKey is the context key for user role
	UserRoleKey = "user_role"
	// UserClaimsKey is the context key for complete claims
	UserClaimsKey = "user_claims"
)

// JWTMiddleware creates a JWT authentication middleware
func JWTMiddleware() gin.HandlerFunc {
	config := jwt.DefaultConfig()

	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized, errorx.KV("msg", "missing authorization header"))))
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrUnauthorized, errorx.KV("msg", "invalid authorization format"))))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := jwt.ParseToken(config, tokenString)
		if err != nil {
			if err == jwt.ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrTokenExpired)))
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, vo.Fail(c, nil, errorx.New(errno.ErrInvalidToken)))
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserNicknameKey, claims.Nickname)
		c.Set(UserRoleKey, claims.Role)
		c.Set(UserClaimsKey, claims)

		c.Next()
	}
}

// OptionalJWTMiddleware creates an optional JWT authentication middleware
// This middleware will parse the token if present, but won't block the request if missing
func OptionalJWTMiddleware() gin.HandlerFunc {
	config := jwt.DefaultConfig()

	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := jwt.ParseToken(config, tokenString)
		if err != nil {
			c.Next()
			return
		}

		// Store user info in context if token is valid
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserNicknameKey, claims.Nickname)
		c.Set(UserRoleKey, claims.Role)
		c.Set(UserClaimsKey, claims)

		c.Next()
	}
}

// GetUserID returns the user ID from context, returns 0 if not authenticated
func GetUserID(c *gin.Context) int64 {
	if userID, exists := c.Get(UserIDKey); exists {
		if id, ok := userID.(int64); ok {
			return id
		}
	}
	return 0
}

// GetUserEmail returns the user email from context
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get(UserEmailKey); exists {
		if e, ok := email.(string); ok {
			return e
		}
	}
	return ""
}

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated(c *gin.Context) bool {
	return GetUserID(c) > 0
}
