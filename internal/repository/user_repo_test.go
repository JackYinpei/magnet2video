package repository

import (
	"context"
	"testing"

	domain "github.com/Done-0/gin-scaffold/domain/user"
)

func newTestUserRepo(t *testing.T) *GormUserRepository {
	dbMgr := setupTestDB(t)
	return NewUserRepository(dbMgr)
}

func TestUserRepo_CreateAndFindByEmail(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u := domain.NewUser("test@example.com", "hashed", "tester")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("ID should be assigned after Create")
	}

	found, err := repo.FindByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if found.Nickname != "tester" {
		t.Errorf("Nickname = %q, want tester", found.Nickname)
	}
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "nope@example.com")
	if err != domain.ErrUserNotFound {
		t.Errorf("err = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_FindByNickname(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u := domain.NewUser("test@example.com", "hashed", "unique_nick")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByNickname(ctx, "unique_nick")
	if err != nil {
		t.Fatalf("FindByNickname failed: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Errorf("Email = %q", found.Email)
	}
}

func TestUserRepo_Save(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u := domain.NewUser("test@example.com", "hashed", "old_nick")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	u.Nickname = "new_nick"
	u.Role = "admin"
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Nickname != "new_nick" {
		t.Errorf("Nickname = %q, want new_nick", found.Nickname)
	}
	if found.Role != "admin" {
		t.Errorf("Role = %q, want admin", found.Role)
	}
}

func TestUserRepo_Delete(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u := domain.NewUser("test@example.com", "hashed", "tester")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.FindByID(ctx, u.ID)
	if err != domain.ErrUserNotFound {
		t.Errorf("err = %v, want ErrUserNotFound after delete", err)
	}
}

func TestUserRepo_List(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := domain.NewUser(
			"user"+string(rune('a'+i))+"@test.com",
			"hashed",
			"user"+string(rune('a'+i)),
		)
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}

	users, total, err := repo.List(ctx, "", "", 1, 3)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(users) != 3 {
		t.Errorf("page size = %d, want 3", len(users))
	}
}

func TestUserRepo_CountAll(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		u := domain.NewUser(
			"user"+string(rune('a'+i))+"@test.com",
			"hashed",
			"user"+string(rune('a'+i)),
		)
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}

	count, err := repo.CountAll(ctx)
	if err != nil {
		t.Fatalf("CountAll failed: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}
