// Package errno provides user-level error code definitions
// Author: Done-0
// Created: 2026-01-22
package errno

import (
	"magnet2video/internal/utils/errorx/code"
)

// User-level error codes: 20000 ~ 29999
// Used: 20001-20008
// Next available: 20009
const (
	ErrUserNotFound       = 20001 // User not found
	ErrUserAlreadyExists  = 20002 // User already exists
	ErrInvalidCredentials = 20003 // Invalid email or password
	ErrInvalidToken       = 20004 // Invalid or expired token
	ErrTokenExpired       = 20005 // Token has expired
	ErrPasswordTooWeak    = 20006 // Password does not meet requirements
	ErrEmailInvalid       = 20007 // Email format is invalid
	ErrNicknameTaken      = 20008 // Nickname is already taken
)

func init() {
	code.Register(ErrUserNotFound, "user not found: {{.msg}}")
	code.Register(ErrUserAlreadyExists, "user already exists: {{.email}}")
	code.Register(ErrInvalidCredentials, "invalid email or password")
	code.Register(ErrInvalidToken, "invalid or expired token")
	code.Register(ErrTokenExpired, "token has expired")
	code.Register(ErrPasswordTooWeak, "password must be at least 6 characters")
	code.Register(ErrEmailInvalid, "invalid email format: {{.email}}")
	code.Register(ErrNicknameTaken, "nickname is already taken: {{.nickname}}")
}
