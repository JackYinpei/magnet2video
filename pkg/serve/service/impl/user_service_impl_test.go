// Package impl provides user service implementation tests
// Author: Done-0
// Created: 2026-02-06
package impl

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	torrentModel "magnet2video/internal/model/torrent"
	userModel "magnet2video/internal/model/user"
	"magnet2video/pkg/serve/controller/dto"
)

// --- Test Setup ---

func setupUserService(t *testing.T) (*UserServiceImpl, *MockDatabaseManager) {
	t.Helper()
	dbMgr := setupTestDB(t)
	logMgr := newMockLoggerManager()
	svc := NewUserService(logMgr, dbMgr).(*UserServiceImpl)
	return svc, dbMgr
}

// --- Register Tests ---

func TestUserService_Register(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, svc *UserServiceImpl, dbMgr *MockDatabaseManager)
		req     *dto.RegisterRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "success",
			req: &dto.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Nickname: "testuser",
			},
			wantErr: false,
		},
		{
			name: "duplicate email",
			setup: func(t *testing.T, svc *UserServiceImpl, dbMgr *MockDatabaseManager) {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
				dbMgr.DB().Create(&userModel.User{
					Email: "dup@example.com", Password: string(hashed),
					Nickname: "existing", Role: "user",
				})
			},
			req: &dto.RegisterRequest{
				Email:    "dup@example.com",
				Password: "password123",
				Nickname: "newuser",
			},
			wantErr: true,
			errMsg:  "email already registered",
		},
		{
			name: "duplicate nickname",
			setup: func(t *testing.T, svc *UserServiceImpl, dbMgr *MockDatabaseManager) {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
				dbMgr.DB().Create(&userModel.User{
					Email: "other@example.com", Password: string(hashed),
					Nickname: "taken", Role: "user",
				})
			},
			req: &dto.RegisterRequest{
				Email:    "new@example.com",
				Password: "password123",
				Nickname: "taken",
			},
			wantErr: true,
			errMsg:  "nickname already taken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupUserService(t)
			if tt.setup != nil {
				tt.setup(t, svc, dbMgr)
			}

			c := newTestGinContext(0)
			resp, err := svc.Register(c, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Register() error = nil, want error containing %q", tt.errMsg)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Register() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("Register() unexpected error: %v", err)
			}

			if resp.User.Email != tt.req.Email {
				t.Errorf("Register() email = %q, want %q", resp.User.Email, tt.req.Email)
			}
			if resp.User.Nickname != tt.req.Nickname {
				t.Errorf("Register() nickname = %q, want %q", resp.User.Nickname, tt.req.Nickname)
			}
			if resp.User.Role != "user" {
				t.Errorf("Register() role = %q, want %q", resp.User.Role, "user")
			}
			if resp.Token == "" {
				t.Error("Register() token is empty")
			}
			if resp.User.ID == 0 {
				t.Error("Register() user ID is 0")
			}

			// Verify password is hashed in DB
			var dbUser userModel.User
			dbMgr.DB().Where("email = ?", tt.req.Email).First(&dbUser)
			if bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(tt.req.Password)) != nil {
				t.Error("Register() stored password does not match bcrypt hash")
			}
		})
	}
}

// --- Login Tests ---

func TestUserService_Login(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		pass    string
		wantErr bool
		errMsg  string
	}{
		{name: "success", email: "login@example.com", pass: "correctpass", wantErr: false},
		{name: "wrong email", email: "noone@example.com", pass: "correctpass", wantErr: true, errMsg: "invalid email or password"},
		{name: "wrong password", email: "login@example.com", pass: "wrongpass", wantErr: true, errMsg: "invalid email or password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupUserService(t)

			// Seed a user
			hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.MinCost)
			dbMgr.DB().Create(&userModel.User{
				Email: "login@example.com", Password: string(hashed),
				Nickname: "loginuser", Role: "user",
			})

			c := newTestGinContext(0)
			resp, err := svc.Login(c, &dto.LoginRequest{Email: tt.email, Password: tt.pass})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Login() error = nil, want error containing %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("Login() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login() unexpected error: %v", err)
			}
			if resp.Token == "" {
				t.Error("Login() token is empty")
			}
			if resp.User.Email != tt.email {
				t.Errorf("Login() email = %q, want %q", resp.User.Email, tt.email)
			}
		})
	}
}

// --- GetProfile Tests ---

func TestUserService_GetProfile(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		wantErr bool
	}{
		{name: "success", userID: 1, wantErr: false},
		{name: "unauthorized - no userID", userID: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupUserService(t)

			// Seed user with known ID
			if tt.userID > 0 {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
				user := &userModel.User{
					Email: "profile@example.com", Password: string(hashed),
					Nickname: "profileuser", Role: "user",
				}
				user.ID = tt.userID
				dbMgr.DB().Create(user)
			}

			c := newTestGinContext(tt.userID)
			resp, err := svc.GetProfile(c)

			if tt.wantErr {
				if err == nil {
					t.Error("GetProfile() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetProfile() unexpected error: %v", err)
			}
			if resp.User.Email != "profile@example.com" {
				t.Errorf("GetProfile() email = %q, want %q", resp.User.Email, "profile@example.com")
			}
		})
	}
}

// --- UpdateProfile Tests ---

func TestUserService_UpdateProfile(t *testing.T) {
	tests := []struct {
		name       string
		req        *dto.UpdateProfileRequest
		seedExtra  bool // seed an extra user with conflicting nickname
		wantErr    bool
		errMsg     string
		wantNick   string
		wantAvatar string
	}{
		{
			name:     "update nickname",
			req:      &dto.UpdateProfileRequest{Nickname: "newnick"},
			wantNick: "newnick",
		},
		{
			name:       "update avatar",
			req:        &dto.UpdateProfileRequest{Avatar: "https://img.example.com/a.png"},
			wantNick:   "origuser",
			wantAvatar: "https://img.example.com/a.png",
		},
		{
			name:      "nickname conflict",
			req:       &dto.UpdateProfileRequest{Nickname: "occupied"},
			seedExtra: true,
			wantErr:   true,
			errMsg:    "nickname already taken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupUserService(t)

			hashed, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
			user := &userModel.User{
				Email: "update@example.com", Password: string(hashed),
				Nickname: "origuser", Role: "user",
			}
			user.ID = 100
			dbMgr.DB().Create(user)

			if tt.seedExtra {
				other := &userModel.User{
					Email: "other@example.com", Password: string(hashed),
					Nickname: "occupied", Role: "user",
				}
				other.ID = 200
				dbMgr.DB().Create(other)
			}

			c := newTestGinContext(int64(100))
			resp, err := svc.UpdateProfile(c, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateProfile() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("UpdateProfile() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateProfile() unexpected error: %v", err)
			}
			if tt.wantNick != "" && resp.User.Nickname != tt.wantNick {
				t.Errorf("UpdateProfile() nickname = %q, want %q", resp.User.Nickname, tt.wantNick)
			}
			if tt.wantAvatar != "" && resp.User.Avatar != tt.wantAvatar {
				t.Errorf("UpdateProfile() avatar = %q, want %q", resp.User.Avatar, tt.wantAvatar)
			}
		})
	}
}

// --- ChangePassword Tests ---

func TestUserService_ChangePassword(t *testing.T) {
	tests := []struct {
		name    string
		oldPass string
		newPass string
		wantErr bool
		errMsg  string
	}{
		{name: "success", oldPass: "oldpass", newPass: "newpass123", wantErr: false},
		{name: "wrong old password", oldPass: "wrongold", newPass: "newpass123", wantErr: true, errMsg: "incorrect current password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupUserService(t)

			hashed, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.MinCost)
			user := &userModel.User{
				Email: "chpwd@example.com", Password: string(hashed),
				Nickname: "chpwduser", Role: "user",
			}
			user.ID = 300
			dbMgr.DB().Create(user)

			c := newTestGinContext(int64(300))
			resp, err := svc.ChangePassword(c, &dto.ChangePasswordRequest{
				OldPassword: tt.oldPass,
				NewPassword: tt.newPass,
			})

			if tt.wantErr {
				if err == nil {
					t.Errorf("ChangePassword() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("ChangePassword() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("ChangePassword() unexpected error: %v", err)
			}
			if resp.Message == "" {
				t.Error("ChangePassword() message is empty")
			}

			// Verify new password works
			var updated userModel.User
			dbMgr.DB().Where("id = ?", 300).First(&updated)
			if bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte(tt.newPass)) != nil {
				t.Error("ChangePassword() new password hash does not match")
			}
		})
	}
}

// --- SetTorrentPublic Tests ---

func TestUserService_SetTorrentPublic(t *testing.T) {
	tests := []struct {
		name       string
		userID     int64
		infoHash   string
		visibility int
		wantErr    bool
		errMsg     string
	}{
		{name: "success - make public", userID: 400, infoHash: "abc123", visibility: 2, wantErr: false},
		{name: "success - make internal", userID: 400, infoHash: "abc123", visibility: 1, wantErr: false},
		{name: "success - make private", userID: 400, infoHash: "abc123", visibility: 0, wantErr: false},
		{name: "not owned by user", userID: 999, infoHash: "abc123", visibility: 2, wantErr: true, errMsg: "torrent not found or not owned by you"},
		{name: "invalid visibility value", userID: 400, infoHash: "abc123", visibility: 3, wantErr: true, errMsg: "invalid visibility value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupUserService(t)

			// Seed torrent owned by user 400
			torrent := &torrentModel.Torrent{
				InfoHash:   "abc123",
				Name:       "test torrent",
				CreatorID:  400,
				Visibility: 0,
			}
			torrent.ID = 1
			dbMgr.DB().Create(torrent)

			c := newTestGinContext(tt.userID)
			resp, err := svc.SetTorrentPublic(c, &dto.SetTorrentPublicRequest{
				InfoHash:   tt.infoHash,
				Visibility: tt.visibility,
			})

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetTorrentPublic() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("SetTorrentPublic() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("SetTorrentPublic() unexpected error: %v", err)
			}
			if resp.Visibility != tt.visibility {
				t.Errorf("SetTorrentPublic() visibility = %d, want %d", resp.Visibility, tt.visibility)
			}

			// Verify DB updated
			var updated torrentModel.Torrent
			dbMgr.DB().Where("info_hash = ?", "abc123").First(&updated)
			if updated.Visibility != tt.visibility {
				t.Errorf("SetTorrentPublic() DB visibility = %d, want %d", updated.Visibility, tt.visibility)
			}
		})
	}
}

// --- Benchmarks ---

func BenchmarkUserService_Register(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupUserService(t)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newTestGinContext(0)
		svc.Register(c, &dto.RegisterRequest{
			Email:    "bench" + string(rune(i+'a')) + "@example.com",
			Password: "password123",
			Nickname: "bench" + string(rune(i+'a')),
		})
	}
}

func BenchmarkUserService_Login(b *testing.B) {
	t := &testing.T{}
	svc, dbMgr := setupUserService(t)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("benchpass"), bcrypt.MinCost)
	dbMgr.DB().Create(&userModel.User{
		Email: "benchlogin@example.com", Password: string(hashed),
		Nickname: "benchlogin", Role: "user",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newTestGinContext(0)
		svc.Login(c, &dto.LoginRequest{
			Email:    "benchlogin@example.com",
			Password: "benchpass",
		})
	}
}
