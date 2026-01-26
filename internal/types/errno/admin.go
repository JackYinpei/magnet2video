// Package errno provides admin-level error code definitions
// Author: Done-0
// Created: 2026-01-26
package errno

import (
	"github.com/Done-0/gin-scaffold/internal/utils/errorx/code"
)

// Admin-level error codes: 40000 ~ 49999
// Used: 40001-40005
// Next available: 40006
const (
	ErrAdminRequired      = 40001 // Admin permission required
	ErrSuperAdminRequired = 40002 // Super admin permission required
	ErrCannotDeleteSelf   = 40003 // Cannot delete own account
	ErrCannotModifySelf   = 40004 // Cannot modify own role
	ErrUserHasResources   = 40005 // User has resources, cannot delete
)

func init() {
	code.Register(ErrAdminRequired, "admin permission required")
	code.Register(ErrSuperAdminRequired, "super admin permission required")
	code.Register(ErrCannotDeleteSelf, "cannot delete your own account")
	code.Register(ErrCannotModifySelf, "cannot modify your own role")
	code.Register(ErrUserHasResources, "user has {{.count}} resources, please delete them first")
}
