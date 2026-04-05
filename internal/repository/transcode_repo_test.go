package repository

import (
	"context"
	"testing"

	domain "github.com/Done-0/gin-scaffold/domain/transcode"
)

func newTestTranscodeRepo(t *testing.T) *GormTranscodeJobRepository {
	dbMgr := setupTestDB(t)
	return NewTranscodeJobRepository(dbMgr)
}

func TestTranscodeRepo_CreateAndFindByID(t *testing.T) {
	repo := newTestTranscodeRepo(t)
	ctx := context.Background()

	job := domain.NewJob(1, "abc123", 0, "/input/video.mkv", "/output/video.mp4", "transcode", 1)
	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("ID should be assigned after Create")
	}

	found, err := repo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.InputPath != "/input/video.mkv" {
		t.Errorf("InputPath = %q", found.InputPath)
	}
	if found.Status != domain.JobPending {
		t.Errorf("Status = %v, want Pending", found.Status)
	}
}

func TestTranscodeRepo_FindByID_NotFound(t *testing.T) {
	repo := newTestTranscodeRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, 9999)
	if err != domain.ErrJobNotFound {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
}

func TestTranscodeRepo_Save(t *testing.T) {
	repo := newTestTranscodeRepo(t)
	ctx := context.Background()

	job := domain.NewJob(1, "abc123", 0, "/input/video.mkv", "/output/video.mp4", "transcode", 1)
	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	job.Status = domain.JobProcessing
	job.Progress = 50
	if err := repo.Save(ctx, job); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := repo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Status != domain.JobProcessing {
		t.Errorf("Status = %v, want Processing", found.Status)
	}
	if found.Progress != 50 {
		t.Errorf("Progress = %d, want 50", found.Progress)
	}
}

func TestTranscodeRepo_FindByTorrentID(t *testing.T) {
	repo := newTestTranscodeRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		job := domain.NewJob(1, "abc123", i, "/input", "/output", "transcode", 1)
		if err := repo.Create(ctx, job); err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}
	// Different torrent
	other := domain.NewJob(2, "def456", 0, "/input", "/output", "remux", 1)
	if err := repo.Create(ctx, other); err != nil {
		t.Fatalf("Create other failed: %v", err)
	}

	jobs, err := repo.FindByTorrentID(ctx, 1)
	if err != nil {
		t.Fatalf("FindByTorrentID failed: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestTranscodeRepo_DeleteByTorrentID(t *testing.T) {
	repo := newTestTranscodeRepo(t)
	ctx := context.Background()

	job := domain.NewJob(1, "abc123", 0, "/input", "/output", "transcode", 1)
	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.DeleteByTorrentID(ctx, 1); err != nil {
		t.Fatalf("DeleteByTorrentID failed: %v", err)
	}

	_, err := repo.FindByID(ctx, job.ID)
	if err != domain.ErrJobNotFound {
		t.Errorf("err = %v, want ErrJobNotFound after delete", err)
	}
}

func TestTranscodeRepo_ListAll(t *testing.T) {
	repo := newTestTranscodeRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		job := domain.NewJob(1, "abc", i, "/input", "/output", "transcode", 1)
		if err := repo.Create(ctx, job); err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}

	jobs, total, err := repo.ListAll(ctx, 1, 3, -1)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(jobs) != 3 {
		t.Errorf("page size = %d, want 3", len(jobs))
	}
}
