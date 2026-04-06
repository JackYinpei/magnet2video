package repository

import (
	"context"
	"testing"

	domain "magnet2video/domain/torrent"
	torrentModel "magnet2video/internal/model/torrent"
)

func newTestTorrentRepo(t *testing.T) (*GormTorrentRepository, *mockDBManager) {
	dbMgr := setupTestDB(t)
	return NewTorrentRepository(dbMgr), dbMgr
}

func TestTorrentRepo_CreateAndFindByInfoHash(t *testing.T) {
	repo, _ := newTestTorrentRepo(t)
	ctx := context.Background()

	tor := domain.NewTorrent("abc123", "Test Torrent", 1024, 1)
	if err := repo.Create(ctx, tor); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if tor.ID == 0 {
		t.Fatal("ID should be assigned after Create")
	}

	found, err := repo.FindByInfoHash(ctx, "abc123")
	if err != nil {
		t.Fatalf("FindByInfoHash failed: %v", err)
	}
	if found.Name != "Test Torrent" {
		t.Errorf("Name = %q, want %q", found.Name, "Test Torrent")
	}
	if found.Status != domain.DownloadPending {
		t.Errorf("Status = %v, want Pending", found.Status)
	}
}

func TestTorrentRepo_FindByInfoHash_NotFound(t *testing.T) {
	repo, _ := newTestTorrentRepo(t)
	ctx := context.Background()

	_, err := repo.FindByInfoHash(ctx, "nonexistent")
	if err != domain.ErrTorrentNotFound {
		t.Errorf("err = %v, want ErrTorrentNotFound", err)
	}
}

func TestTorrentRepo_SaveUpdatesFields(t *testing.T) {
	repo, _ := newTestTorrentRepo(t)
	ctx := context.Background()

	tor := domain.NewTorrent("abc123", "Original", 1024, 1)
	if err := repo.Create(ctx, tor); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	tor.Name = "Updated"
	tor.Status = domain.DownloadDownloading
	tor.Progress = 50.5
	if err := repo.Save(ctx, tor); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := repo.FindByID(ctx, tor.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Updated" {
		t.Errorf("Name = %q, want Updated", found.Name)
	}
	if found.Status != domain.DownloadDownloading {
		t.Errorf("Status = %v, want Downloading", found.Status)
	}
}

func TestTorrentRepo_FindByIDWithFiles(t *testing.T) {
	repo, dbMgr := newTestTorrentRepo(t)
	ctx := context.Background()

	tor := domain.NewTorrent("abc123", "Test", 1024, 1)
	if err := repo.Create(ctx, tor); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Insert files directly via GORM
	files := []torrentModel.TorrentFile{
		{TorrentID: tor.ID, Index: 0, Path: "video.mp4", Type: "video", Source: "original", IsSelected: true},
		{TorrentID: tor.ID, Index: 1, Path: "sub.srt", Type: "subtitle", Source: "extracted", IsSelected: true},
	}
	for i := range files {
		if err := dbMgr.DB().Create(&files[i]).Error; err != nil {
			t.Fatalf("create file failed: %v", err)
		}
	}

	found, err := repo.FindByIDWithFiles(ctx, tor.ID)
	if err != nil {
		t.Fatalf("FindByIDWithFiles failed: %v", err)
	}
	if len(found.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(found.Files))
	}
	if found.Files[0].Type != domain.FileTypeVideo {
		t.Errorf("file[0].Type = %q, want video", found.Files[0].Type)
	}
}

func TestTorrentRepo_ListByCreator_Pagination(t *testing.T) {
	repo, _ := newTestTorrentRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tor := domain.NewTorrent("hash"+string(rune('a'+i)), "Torrent", 1024, 1)
		if err := repo.Create(ctx, tor); err != nil {
			t.Fatalf("Create #%d failed: %v", i, err)
		}
	}
	// Different creator
	other := domain.NewTorrent("other", "Other", 1024, 2)
	if err := repo.Create(ctx, other); err != nil {
		t.Fatalf("Create other failed: %v", err)
	}

	torrents, total, err := repo.ListByCreator(ctx, 1, 1, 3)
	if err != nil {
		t.Fatalf("ListByCreator failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(torrents) != 3 {
		t.Errorf("page size = %d, want 3", len(torrents))
	}
}

func TestTorrentRepo_Delete(t *testing.T) {
	repo, _ := newTestTorrentRepo(t)
	ctx := context.Background()

	tor := domain.NewTorrent("abc123", "Test", 1024, 1)
	if err := repo.Create(ctx, tor); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, tor.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.FindByID(ctx, tor.ID)
	if err != domain.ErrTorrentNotFound {
		t.Errorf("err = %v, want ErrTorrentNotFound after delete", err)
	}
}

func TestTorrentRepo_FindActiveForRestore(t *testing.T) {
	repo, _ := newTestTorrentRepo(t)
	ctx := context.Background()

	downloading := domain.NewTorrent("dl", "Downloading", 1024, 1)
	downloading.Status = domain.DownloadDownloading
	if err := repo.Create(ctx, downloading); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	paused := domain.NewTorrent("ps", "Paused", 1024, 1)
	paused.Status = domain.DownloadPaused
	if err := repo.Create(ctx, paused); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	completed := domain.NewTorrent("cp", "Completed", 1024, 1)
	completed.Status = domain.DownloadCompleted
	if err := repo.Create(ctx, completed); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Need to save the status since Create hooks may override
	repo.db().Model(&torrentModel.Torrent{}).Where("info_hash = ?", "dl").Update("status", int(domain.DownloadDownloading))
	repo.db().Model(&torrentModel.Torrent{}).Where("info_hash = ?", "ps").Update("status", int(domain.DownloadPaused))
	repo.db().Model(&torrentModel.Torrent{}).Where("info_hash = ?", "cp").Update("status", int(domain.DownloadCompleted))

	active, err := repo.FindActiveForRestore(ctx)
	if err != nil {
		t.Fatalf("FindActiveForRestore failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active torrents, got %d", len(active))
	}
}

func TestTorrentRepo_UpdateFileFields(t *testing.T) {
	repo, dbMgr := newTestTorrentRepo(t)
	ctx := context.Background()

	tor := domain.NewTorrent("abc123", "Test", 1024, 1)
	if err := repo.Create(ctx, tor); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	file := torrentModel.TorrentFile{
		TorrentID:         tor.ID,
		Index:             0,
		Path:              "video.mp4",
		Type:              "video",
		CloudUploadStatus: 0,
	}
	if err := dbMgr.DB().Create(&file).Error; err != nil {
		t.Fatalf("create file failed: %v", err)
	}

	err := repo.UpdateFileFields(ctx, tor.ID, 0, map[string]interface{}{
		"cloud_upload_status": 3,
		"cloud_path":          "torrents/abc123/video.mp4",
	})
	if err != nil {
		t.Fatalf("UpdateFileFields failed: %v", err)
	}

	var updated torrentModel.TorrentFile
	dbMgr.DB().Where("torrent_id = ? AND `index` = ?", tor.ID, 0).First(&updated)
	if updated.CloudUploadStatus != 3 {
		t.Errorf("CloudUploadStatus = %d, want 3", updated.CloudUploadStatus)
	}
	if updated.CloudPath != "torrents/abc123/video.mp4" {
		t.Errorf("CloudPath = %q", updated.CloudPath)
	}
}
