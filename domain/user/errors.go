// Package user provides the user bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package user

import "errors"

var (
	// ErrCannotDeleteSuperAdmin prevents deletion of super admin accounts.
	ErrCannotDeleteSuperAdmin = errors.New("cannot delete super admin")

	// ErrCannotDeleteSelf prevents users from deleting themselves.
	ErrCannotDeleteSelf = errors.New("cannot delete self")

	// ErrUserNotFound indicates the user does not exist.
	ErrUserNotFound = errors.New("user not found")
)
