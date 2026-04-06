// Package auth provides admin authentication middleware
// Author: Done-0
// Created: 2026-01-26
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"magnet2video/internal/types/errno"
	"magnet2video/internal/utils/errorx"
	"magnet2video/internal/utils/jwt"
	"magnet2video/internal/utils/vo"
)

// AdminMiddleware creates a middleware that requires admin role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(UserRoleKey)
		if !exists {
			c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrAdminRequired)))
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok || roleStr != "admin" {
			c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrAdminRequired)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// SuperAdminMiddleware creates a middleware that requires super admin
func SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get(UserClaimsKey)
		if !exists {
			c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrSuperAdminRequired)))
			c.Abort()
			return
		}

		jwtClaims, ok := claims.(*jwt.Claims)
		if !ok {
			c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrSuperAdminRequired)))
			c.Abort()
			return
		}

		// Check if the user is super admin
		// Note: IsSuperAdmin flag should be checked from database in actual implementation
		// Here we rely on the role being "admin" as a basic check
		if jwtClaims.Role != "admin" {
			c.JSON(http.StatusForbidden, vo.Fail(c, nil, errorx.New(errno.ErrSuperAdminRequired)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserRole returns the user role from context
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get(UserRoleKey); exists {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

// IsAdmin checks if the user is an admin
func IsAdmin(c *gin.Context) bool {
	return GetUserRole(c) == "admin"
}
