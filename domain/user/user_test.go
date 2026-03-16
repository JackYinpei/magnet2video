package user

import "testing"

func TestNewUser(t *testing.T) {
	u := NewUser("test@example.com", "hashed", "tester")
	if u.Role != "user" {
		t.Errorf("role = %q, want user", u.Role)
	}
	if u.Email != "test@example.com" {
		t.Errorf("email = %q", u.Email)
	}
}

func TestUser_IsAdmin(t *testing.T) {
	u := NewUser("a@b.com", "p", "nick")
	if u.IsAdmin() {
		t.Error("regular user should not be admin")
	}
	u.Role = "admin"
	if !u.IsAdmin() {
		t.Error("admin role should be admin")
	}
	u.Role = "user"
	u.IsSuperAdmin = true
	if !u.IsAdmin() {
		t.Error("super admin should be admin")
	}
}

func TestUser_CanBeDeletedBy(t *testing.T) {
	u := &User{ID: 1, IsSuperAdmin: true}
	if err := u.CanBeDeletedBy(2); err != ErrCannotDeleteSuperAdmin {
		t.Errorf("err = %v, want ErrCannotDeleteSuperAdmin", err)
	}

	u.IsSuperAdmin = false
	if err := u.CanBeDeletedBy(1); err != ErrCannotDeleteSelf {
		t.Errorf("err = %v, want ErrCannotDeleteSelf", err)
	}

	if err := u.CanBeDeletedBy(2); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

func TestUser_PromoteToAdmin(t *testing.T) {
	u := NewUser("a@b.com", "p", "nick")
	u.PromoteToAdmin()
	if u.Role != "admin" {
		t.Errorf("role = %q, want admin", u.Role)
	}
}

func TestUser_UpdateProfile(t *testing.T) {
	u := NewUser("a@b.com", "p", "old")
	u.UpdateProfile("new", "avatar.png")
	if u.Nickname != "new" {
		t.Errorf("nickname = %q, want new", u.Nickname)
	}
	if u.Avatar != "avatar.png" {
		t.Errorf("avatar = %q", u.Avatar)
	}
}

func TestUser_UpdateProfile_Empty(t *testing.T) {
	u := NewUser("a@b.com", "p", "old")
	u.Avatar = "old.png"
	u.UpdateProfile("", "")
	if u.Nickname != "old" {
		t.Error("empty nickname should not update")
	}
	if u.Avatar != "old.png" {
		t.Error("empty avatar should not update")
	}
}
