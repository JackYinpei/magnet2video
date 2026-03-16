// Package user provides the user bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package user

import "context"

// UserRepository abstracts persistence for the User aggregate.
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByNickname(ctx context.Context, nickname string) (*User, error)
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, search, role string, page, pageSize int) ([]User, int64, error)
	CountAll(ctx context.Context) (int64, error)
}
