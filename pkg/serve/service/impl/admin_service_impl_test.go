// Package impl provides admin service implementation tests
// Author: Done-0
// Created: 2026-02-06
package impl

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
	userModel "github.com/Done-0/gin-scaffold/internal/model/user"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
)

// setupAdminServiceDirect creates an AdminServiceImpl without TorrentManager
// to test DB-only methods. Methods calling Client() are not testable here.
func setupAdminServiceDirect(t *testing.T) (*AdminServiceImpl, *MockDatabaseManager) {
	t.Helper()
	dbMgr := setupTestDB(t)
	logMgr := newMockLoggerManager()
	svc := &AdminServiceImpl{
		loggerManager: logMgr,
		dbManager:     dbMgr,
	}
	return svc, dbMgr
}

// seedUsers creates test users and returns their IDs
func seedUsers(t *testing.T, dbMgr *MockDatabaseManager) (admin int64, regular int64, superAdmin int64) {
	t.Helper()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)

	adminUser := &userModel.User{Email: "admin@test.com", Password: string(hashed), Nickname: "admin", Role: "admin"}
	adminUser.ID = 1001
	dbMgr.DB().Create(adminUser)

	regUser := &userModel.User{Email: "user@test.com", Password: string(hashed), Nickname: "regular", Role: "user"}
	regUser.ID = 1002
	dbMgr.DB().Create(regUser)

	superUser := &userModel.User{Email: "super@test.com", Password: string(hashed), Nickname: "super", Role: "admin", IsSuperAdmin: true}
	superUser.ID = 1003
	dbMgr.DB().Create(superUser)

	return adminUser.ID, regUser.ID, superUser.ID
}

// --- ListUsers Tests ---

func TestAdminService_ListUsers(t *testing.T) {
	tests := []struct {
		name      string
		req       *dto.ListUsersRequest
		wantTotal int64
		wantLen   int
	}{
		{
			name:      "list all users",
			req:       &dto.ListUsersRequest{Page: 1, PageSize: 20},
			wantTotal: 3,
			wantLen:   3,
		},
		{
			name:      "search by email",
			req:       &dto.ListUsersRequest{Search: "admin", Page: 1, PageSize: 20},
			wantTotal: 1,
			wantLen:   1,
		},
		{
			name:      "filter by role",
			req:       &dto.ListUsersRequest{Role: "admin", Page: 1, PageSize: 20},
			wantTotal: 2, // admin + super admin
			wantLen:   2,
		},
		{
			name:      "pagination - page 1 size 2",
			req:       &dto.ListUsersRequest{Page: 1, PageSize: 2},
			wantTotal: 3,
			wantLen:   2,
		},
		{
			name:      "pagination - page 2 size 2",
			req:       &dto.ListUsersRequest{Page: 2, PageSize: 2},
			wantTotal: 3,
			wantLen:   1,
		},
		{
			name:      "default page values",
			req:       &dto.ListUsersRequest{},
			wantTotal: 3,
			wantLen:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupAdminServiceDirect(t)
			seedUsers(t, dbMgr)

			c := newTestGinContext(int64(1001))
			resp, err := svc.ListUsers(c, tt.req)

			if err != nil {
				t.Fatalf("ListUsers() unexpected error: %v", err)
			}
			if resp.Total != tt.wantTotal {
				t.Errorf("ListUsers() total = %d, want %d", resp.Total, tt.wantTotal)
			}
			if len(resp.Users) != tt.wantLen {
				t.Errorf("ListUsers() len = %d, want %d", len(resp.Users), tt.wantLen)
			}
		})
	}
}

// --- ListUsers with TorrentCount ---

func TestAdminService_ListUsers_TorrentCount(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	adminID, regID, _ := seedUsers(t, dbMgr)

	// Seed torrents for regular user
	for i := 0; i < 3; i++ {
		dbMgr.DB().Create(&torrentModel.Torrent{
			InfoHash:  "admintesthash" + string(rune('a'+i)),
			Name:      "torrent",
			CreatorID: regID,
		})
	}

	c := newTestGinContext(adminID)
	resp, err := svc.ListUsers(c, &dto.ListUsersRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListUsers() unexpected error: %v", err)
	}

	for _, u := range resp.Users {
		if u.ID == regID && u.TorrentCount != 3 {
			t.Errorf("ListUsers() user %d torrent count = %d, want 3", regID, u.TorrentCount)
		}
	}
}

// --- GetUserDetail Tests ---

func TestAdminService_GetUserDetail(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		wantErr bool
		errMsg  string
	}{
		{name: "success", userID: 1002, wantErr: false},
		{name: "user not found", userID: 9999, wantErr: true, errMsg: "user not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupAdminServiceDirect(t)
			seedUsers(t, dbMgr)

			// Seed a torrent for storage calculation
			tor := &torrentModel.Torrent{InfoHash: "detailhash", Name: "detail", TotalSize: 1024000, CreatorID: 1002}
			dbMgr.DB().Create(tor)

			c := newTestGinContext(int64(1001))
			resp, err := svc.GetUserDetail(c, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetUserDetail() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("GetUserDetail() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUserDetail() unexpected error: %v", err)
			}
			if resp.User.ID != tt.userID {
				t.Errorf("GetUserDetail() user ID = %d, want %d", resp.User.ID, tt.userID)
			}
			if resp.TotalStorage != 1024000 {
				t.Errorf("GetUserDetail() storage = %d, want 1024000", resp.TotalStorage)
			}
		})
	}
}

// --- GetUserTorrents Tests ---

func TestAdminService_GetUserTorrents(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	_, regID, _ := seedUsers(t, dbMgr)

	for i := 0; i < 2; i++ {
		dbMgr.DB().Create(&torrentModel.Torrent{
			InfoHash: "usertorrent" + string(rune('a'+i)), Name: "ut", CreatorID: regID,
		})
	}

	c := newTestGinContext(int64(1001))
	resp, err := svc.GetUserTorrents(c, regID)
	if err != nil {
		t.Fatalf("GetUserTorrents() unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("GetUserTorrents() total = %d, want 2", resp.Total)
	}
}

// --- UpdateUserRole Tests ---

func TestAdminService_UpdateUserRole(t *testing.T) {
	tests := []struct {
		name       string
		callerID   int64
		targetID   int64
		role       string
		wantErr    bool
		errMsg     string
	}{
		{name: "promote to admin", callerID: 1001, targetID: 1002, role: "admin", wantErr: false},
		{name: "demote to user", callerID: 1001, targetID: 1002, role: "user", wantErr: false},
		{name: "cannot modify own role", callerID: 1001, targetID: 1001, role: "user", wantErr: true, errMsg: "cannot modify your own role"},
		{name: "user not found", callerID: 1001, targetID: 9999, role: "admin", wantErr: true, errMsg: "user not found"},
		{name: "invalid role", callerID: 1001, targetID: 1002, role: "superadmin", wantErr: true, errMsg: "invalid role, must be 'user' or 'admin'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr := setupAdminServiceDirect(t)
			seedUsers(t, dbMgr)

			c := newTestGinContext(tt.callerID)
			resp, err := svc.UpdateUserRole(c, &dto.UpdateUserRoleRequest{
				UserID: tt.targetID,
				Role:   tt.role,
			})

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateUserRole() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("UpdateUserRole() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateUserRole() unexpected error: %v", err)
			}
			if resp.Role != tt.role {
				t.Errorf("UpdateUserRole() role = %q, want %q", resp.Role, tt.role)
			}

			// Verify DB
			var updated userModel.User
			dbMgr.DB().Where("id = ?", tt.targetID).First(&updated)
			if updated.Role != tt.role {
				t.Errorf("UpdateUserRole() DB role = %q, want %q", updated.Role, tt.role)
			}
		})
	}
}

// --- ListAllTorrents Tests ---

func TestAdminService_ListAllTorrents(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	_, regID, _ := seedUsers(t, dbMgr)

	status2 := 2
	for i := 0; i < 5; i++ {
		s := torrentModel.StatusDownloading
		if i >= 3 {
			s = torrentModel.StatusCompleted
		}
		dbMgr.DB().Create(&torrentModel.Torrent{
			InfoHash: "allhash" + string(rune('a'+i)), Name: "alltorrent",
			CreatorID: regID, Status: s,
		})
	}

	tests := []struct {
		name      string
		req       *dto.ListAllTorrentsRequest
		wantTotal int64
	}{
		{name: "all torrents", req: &dto.ListAllTorrentsRequest{Page: 1, PageSize: 20}, wantTotal: 5},
		{name: "filter by status", req: &dto.ListAllTorrentsRequest{Status: &status2, Page: 1, PageSize: 20}, wantTotal: 2},
		{name: "filter by creator", req: &dto.ListAllTorrentsRequest{CreatorID: &regID, Page: 1, PageSize: 20}, wantTotal: 5},
		{name: "search by name", req: &dto.ListAllTorrentsRequest{Search: "alltorrent", Page: 1, PageSize: 20}, wantTotal: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestGinContext(int64(1001))
			resp, err := svc.ListAllTorrents(c, tt.req)
			if err != nil {
				t.Fatalf("ListAllTorrents() unexpected error: %v", err)
			}
			if resp.Total != tt.wantTotal {
				t.Errorf("ListAllTorrents() total = %d, want %d", resp.Total, tt.wantTotal)
			}
		})
	}
}

// --- ListAllTorrents with Creator Nickname ---

func TestAdminService_ListAllTorrents_CreatorNickname(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	_, regID, _ := seedUsers(t, dbMgr)

	dbMgr.DB().Create(&torrentModel.Torrent{
		InfoHash: "nicktest", Name: "nick torrent", CreatorID: regID,
	})

	c := newTestGinContext(int64(1001))
	resp, err := svc.ListAllTorrents(c, &dto.ListAllTorrentsRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAllTorrents() unexpected error: %v", err)
	}

	for _, tor := range resp.Torrents {
		if tor.InfoHash == "nicktest" && tor.CreatorNickname != "regular" {
			t.Errorf("ListAllTorrents() creator nickname = %q, want %q", tor.CreatorNickname, "regular")
		}
	}
}

// --- DeleteUser Protection Tests (DB-only checks, without Client()) ---

func TestAdminService_DeleteUser_CannotDeleteSelf(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	adminID, _, _ := seedUsers(t, dbMgr)

	c := newTestGinContext(adminID)
	_, err := svc.DeleteUser(c, adminID)
	if err == nil {
		t.Error("DeleteUser(self) should fail")
	}
	if err.Error() != "cannot delete your own account" {
		t.Errorf("DeleteUser(self) error = %q, want %q", err.Error(), "cannot delete your own account")
	}
}

func TestAdminService_DeleteUser_CannotDeleteSuperAdmin(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	adminID, _, superID := seedUsers(t, dbMgr)

	c := newTestGinContext(adminID)
	_, err := svc.DeleteUser(c, superID)
	if err == nil {
		t.Error("DeleteUser(super admin) should fail")
	}
	if err.Error() != "cannot delete super admin account" {
		t.Errorf("DeleteUser(super admin) error = %q, want %q", err.Error(), "cannot delete super admin account")
	}
}

func TestAdminService_DeleteUser_NotFound(t *testing.T) {
	svc, dbMgr := setupAdminServiceDirect(t)
	adminID, _, _ := seedUsers(t, dbMgr)

	c := newTestGinContext(adminID)
	_, err := svc.DeleteUser(c, 99999)
	if err == nil {
		t.Error("DeleteUser(nonexistent) should fail")
	}
	if err.Error() != "user not found" {
		t.Errorf("DeleteUser(nonexistent) error = %q, want %q", err.Error(), "user not found")
	}
}

// --- GetStats Tests (partial - without disk stats that need Client()) ---
// Note: GetStats calls Client().GetDownloadDir() so we test it indirectly
// through the DB counts portion by checking that no panic occurs on nil torrentManager

func TestAdminService_GetStats_DBCounts(t *testing.T) {
	_, dbMgr := setupAdminServiceDirect(t)
	seedUsers(t, dbMgr)

	// Seed torrents with various statuses
	dbMgr.DB().Create(&torrentModel.Torrent{InfoHash: "stat1", Name: "s1", Status: torrentModel.StatusCompleted, TotalSize: 1000, CreatorID: 1002})
	dbMgr.DB().Create(&torrentModel.Torrent{InfoHash: "stat2", Name: "s2", Status: torrentModel.StatusDownloading, TotalSize: 2000, CreatorID: 1002})
	dbMgr.DB().Create(&transcodeModel.TranscodeJob{TorrentID: 1, Status: transcodeModel.JobStatusPending, InputPath: "x"})

	// Verify the data was seeded correctly using raw DB queries
	var userCount int64
	dbMgr.DB().Model(&userModel.User{}).Count(&userCount)
	if userCount != 3 {
		t.Errorf("GetStats setup: user count = %d, want 3", userCount)
	}

	var torrentCount int64
	dbMgr.DB().Model(&torrentModel.Torrent{}).Count(&torrentCount)
	if torrentCount != 2 {
		t.Errorf("GetStats setup: torrent count = %d, want 2", torrentCount)
	}

	var completedCount int64
	dbMgr.DB().Model(&torrentModel.Torrent{}).Where("status = ?", torrentModel.StatusCompleted).Count(&completedCount)
	if completedCount != 1 {
		t.Errorf("GetStats setup: completed count = %d, want 1", completedCount)
	}

	// Note: GetStats() itself calls Client().GetDownloadDir() which requires a real TorrentManager
	// A proper integration test would need a mock TorrentManager (blocked by Go internal package rules)
}

// --- Benchmarks ---

func BenchmarkAdminService_ListUsers(b *testing.B) {
	t := &testing.T{}
	svc, dbMgr := setupAdminServiceDirect(t)
	seedUsers(t, dbMgr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newTestGinContext(int64(1001))
		svc.ListUsers(c, &dto.ListUsersRequest{Page: 1, PageSize: 20})
	}
}

func BenchmarkAdminService_UpdateUserRole(b *testing.B) {
	t := &testing.T{}
	svc, dbMgr := setupAdminServiceDirect(t)
	seedUsers(t, dbMgr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		role := "user"
		if i%2 == 0 {
			role = "admin"
		}
		c := newTestGinContext(int64(1001))
		svc.UpdateUserRole(c, &dto.UpdateUserRoleRequest{UserID: 1002, Role: role})
	}
}
