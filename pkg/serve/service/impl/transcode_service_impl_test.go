// Package impl provides transcode service implementation tests
// Author: Done-0
// Created: 2026-02-06
package impl

import (
	"testing"

	"github.com/Done-0/gin-scaffold/configs"
	torrentModel "github.com/Done-0/gin-scaffold/internal/model/torrent"
	transcodeModel "github.com/Done-0/gin-scaffold/internal/model/transcode"
	"github.com/Done-0/gin-scaffold/pkg/serve/controller/dto"
)

// setupTranscodeServiceDirect creates a TranscodeServiceImpl without calling the constructor
// to avoid FFmpeg dependency. Only DB-centric methods are testable.
func setupTranscodeServiceDirect(t *testing.T) (*TranscodeServiceImpl, *MockDatabaseManager, *MockQueueProducer) {
	t.Helper()
	dbMgr := setupTestDB(t)
	logMgr := newMockLoggerManager()
	queueProducer := newMockQueueProducer()
	svc := &TranscodeServiceImpl{
		config:        &configs.Config{},
		loggerManager: logMgr,
		dbManager:     dbMgr,
		queueProducer: queueProducer,
	}
	return svc, dbMgr, queueProducer
}

// seedTranscodeData creates a torrent with files and transcode jobs for testing
func seedTranscodeData(t *testing.T, dbMgr *MockDatabaseManager) (torrentID int64, failedJobID int64, pendingJobID int64) {
	t.Helper()

	torrent := &torrentModel.Torrent{
		InfoHash:        "transcode-test-hash",
		Name:            "transcode test",
		Status:          torrentModel.StatusCompleted,
		CreatorID:       500,
		TranscodeStatus: torrentModel.TranscodeStatusPending,
		TotalTranscode:  2,
		Files: torrentModel.TorrentFiles{
			{Path: "video1.mkv", Size: 1000000, IsSelected: true, IsStreamable: true, Type: "video", Source: "original", TranscodeStatus: torrentModel.TranscodeStatusFailed},
			{Path: "video2.mkv", Size: 2000000, IsSelected: true, IsStreamable: true, Type: "video", Source: "original", TranscodeStatus: torrentModel.TranscodeStatusPending},
			{Path: "sub.srt", Size: 5000, IsSelected: true, Type: "subtitle", Source: "original"},
		},
	}
	torrent.ID = 2001
	dbMgr.DB().Create(torrent)

	failedJob := &transcodeModel.TranscodeJob{
		TorrentID:     torrent.ID,
		InfoHash:      torrent.InfoHash,
		InputPath:     "/data/download/video1.mkv",
		OutputPath:    "/data/download/video1_transcoded.mp4",
		FileIndex:     0,
		Status:        transcodeModel.JobStatusFailed,
		InputCodec:    "hevc",
		OutputCodec:   "h264",
		TranscodeType: "transcode",
		ErrorMessage:  "ffmpeg error",
		CreatorID:     500,
	}
	failedJob.ID = 3001
	dbMgr.DB().Create(failedJob)

	pendingJob := &transcodeModel.TranscodeJob{
		TorrentID:     torrent.ID,
		InfoHash:      torrent.InfoHash,
		InputPath:     "/data/download/video2.mkv",
		OutputPath:    "/data/download/video2_transcoded.mp4",
		FileIndex:     1,
		Status:        transcodeModel.JobStatusPending,
		InputCodec:    "hevc",
		OutputCodec:   "h264",
		TranscodeType: "transcode",
		CreatorID:     500,
	}
	pendingJob.ID = 3002
	dbMgr.DB().Create(pendingJob)

	return torrent.ID, failedJob.ID, pendingJob.ID
}

// --- GetTranscodeStatus Tests ---

func TestTranscodeService_GetTranscodeStatus(t *testing.T) {
	tests := []struct {
		name          string
		torrentID     int64
		wantErr       bool
		errMsg        string
		wantJobCount  int
		wantFileCount int
	}{
		{name: "success", torrentID: 2001, wantErr: false, wantJobCount: 2, wantFileCount: 3},
		{name: "torrent not found", torrentID: 9999, wantErr: true, errMsg: "torrent not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr, _ := setupTranscodeServiceDirect(t)
			seedTranscodeData(t, dbMgr)

			c := newTestGinContext(int64(500))
			resp, err := svc.GetTranscodeStatus(c, tt.torrentID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTranscodeStatus() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("GetTranscodeStatus() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetTranscodeStatus() unexpected error: %v", err)
			}
			if resp.TorrentID != tt.torrentID {
				t.Errorf("GetTranscodeStatus() torrentID = %d, want %d", resp.TorrentID, tt.torrentID)
			}
			if len(resp.Jobs) != tt.wantJobCount {
				t.Errorf("GetTranscodeStatus() jobs count = %d, want %d", len(resp.Jobs), tt.wantJobCount)
			}
			// Files should include only original video/subtitle files (excluding non-original)
			if len(resp.Files) != tt.wantFileCount {
				t.Errorf("GetTranscodeStatus() files count = %d, want %d", len(resp.Files), tt.wantFileCount)
			}
			if resp.OverallStatus != torrentModel.TranscodeStatusPending {
				t.Errorf("GetTranscodeStatus() overall status = %d, want %d", resp.OverallStatus, torrentModel.TranscodeStatusPending)
			}
		})
	}
}

// --- RetryTranscode Tests ---

func TestTranscodeService_RetryTranscode(t *testing.T) {
	tests := []struct {
		name    string
		jobID   int64
		wantErr bool
		errMsg  string
	}{
		{name: "success - retry failed job", jobID: 3001, wantErr: false},
		{name: "cannot retry non-failed job", jobID: 3002, wantErr: true, errMsg: "can only retry failed jobs"},
		{name: "job not found", jobID: 9999, wantErr: true, errMsg: "job not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr, queueProducer := setupTranscodeServiceDirect(t)
			seedTranscodeData(t, dbMgr)

			c := newTestGinContext(int64(500))
			resp, err := svc.RetryTranscode(c, &dto.RetryTranscodeRequest{JobID: tt.jobID})

			if tt.wantErr {
				if err == nil {
					t.Errorf("RetryTranscode() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("RetryTranscode() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("RetryTranscode() unexpected error: %v", err)
			}
			if resp.JobID == 0 {
				t.Error("RetryTranscode() new job ID is 0")
			}
			if resp.JobID == tt.jobID {
				t.Error("RetryTranscode() should create a NEW job, not reuse old ID")
			}

			// Verify a new job was created in DB
			var newJob transcodeModel.TranscodeJob
			if err := dbMgr.DB().Where("id = ?", resp.JobID).First(&newJob).Error; err != nil {
				t.Fatalf("RetryTranscode() new job not found in DB: %v", err)
			}
			if newJob.Status != transcodeModel.JobStatusPending {
				t.Errorf("RetryTranscode() new job status = %d, want %d", newJob.Status, transcodeModel.JobStatusPending)
			}

			// Verify message was sent to queue
			if queueProducer.len() != 1 {
				t.Errorf("RetryTranscode() queue messages = %d, want 1", queueProducer.len())
			}

			// Verify torrent file status was reset to pending
			var torrent torrentModel.Torrent
			dbMgr.DB().Where("id = ?", 2001).First(&torrent)
			if torrent.Files[0].TranscodeStatus != torrentModel.TranscodeStatusPending {
				t.Errorf("RetryTranscode() file transcode status = %d, want %d",
					torrent.Files[0].TranscodeStatus, torrentModel.TranscodeStatusPending)
			}
		})
	}
}

// --- CancelTranscode Tests ---

func TestTranscodeService_CancelTranscode(t *testing.T) {
	tests := []struct {
		name    string
		jobID   int64
		wantErr bool
		errMsg  string
	}{
		{name: "cancel pending job", jobID: 3002, wantErr: false},
		{name: "cannot cancel failed job", jobID: 3001, wantErr: true, errMsg: "can only cancel pending or processing jobs"},
		{name: "job not found", jobID: 9999, wantErr: true, errMsg: "job not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dbMgr, _ := setupTranscodeServiceDirect(t)
			seedTranscodeData(t, dbMgr)

			c := newTestGinContext(int64(500))
			resp, err := svc.CancelTranscode(c, tt.jobID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CancelTranscode() error = nil, want %q", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("CancelTranscode() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("CancelTranscode() unexpected error: %v", err)
			}
			if resp.JobID != tt.jobID {
				t.Errorf("CancelTranscode() jobID = %d, want %d", resp.JobID, tt.jobID)
			}

			// Verify job status in DB
			var job transcodeModel.TranscodeJob
			dbMgr.DB().Where("id = ?", tt.jobID).First(&job)
			if job.Status != transcodeModel.JobStatusFailed {
				t.Errorf("CancelTranscode() DB job status = %d, want %d (failed/canceled)", job.Status, transcodeModel.JobStatusFailed)
			}
			if job.ErrorMessage != "canceled by user" {
				t.Errorf("CancelTranscode() DB error message = %q, want %q", job.ErrorMessage, "canceled by user")
			}

			// Verify torrent file status was updated
			var torrent torrentModel.Torrent
			dbMgr.DB().Where("id = ?", 2001).First(&torrent)
			if torrent.Files[1].TranscodeStatus != torrentModel.TranscodeStatusFailed {
				t.Errorf("CancelTranscode() file transcode status = %d, want %d",
					torrent.Files[1].TranscodeStatus, torrentModel.TranscodeStatusFailed)
			}
		})
	}
}

// --- Benchmarks ---

func BenchmarkTranscodeService_GetTranscodeStatus(b *testing.B) {
	t := &testing.T{}
	svc, dbMgr, _ := setupTranscodeServiceDirect(t)
	seedTranscodeData(t, dbMgr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newTestGinContext(int64(500))
		svc.GetTranscodeStatus(c, 2001)
	}
}

func BenchmarkTranscodeService_RetryTranscode(b *testing.B) {
	// Each iteration needs fresh data since retry creates new records
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		t := &testing.T{}
		svc, dbMgr, _ := setupTranscodeServiceDirect(t)
		seedTranscodeData(t, dbMgr)
		b.StartTimer()

		c := newTestGinContext(int64(500))
		svc.RetryTranscode(c, &dto.RetryTranscodeRequest{JobID: 3001})
	}
}
