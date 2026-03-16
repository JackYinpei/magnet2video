// Package user provides the user bounded context domain model.
// Author: Done-0
// Created: 2026-03-16
package user

// User is the aggregate root for a user account.
type User struct {
	ID           int64
	Email        string
	Password     string
	Nickname     string
	Avatar       string
	Role         string
	IsSuperAdmin bool
	CreatedAt    int64
	UpdatedAt    int64
}

// NewUser creates a new user with default role.
func NewUser(email, password, nickname string) *User {
	return &User{
		Email:    email,
		Password: password,
		Nickname: nickname,
		Role:     "user",
	}
}

// IsAdmin returns true if the user has admin privileges.
func (u *User) IsAdmin() bool {
	return u.Role == "admin" || u.IsSuperAdmin
}

// CanBeDeletedBy checks whether this user can be deleted by the given operator.
func (u *User) CanBeDeletedBy(operatorID int64) error {
	if u.IsSuperAdmin {
		return ErrCannotDeleteSuperAdmin
	}
	if u.ID == operatorID {
		return ErrCannotDeleteSelf
	}
	return nil
}

// PromoteToAdmin sets the user role to admin.
func (u *User) PromoteToAdmin() {
	u.Role = "admin"
}

// UpdateProfile updates nickname and avatar.
func (u *User) UpdateProfile(nickname, avatar string) {
	if nickname != "" {
		u.Nickname = nickname
	}
	if avatar != "" {
		u.Avatar = avatar
	}
}
