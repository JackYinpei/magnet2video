package transcode

import "testing"

func newTestJob() *Job {
	return NewJob(1, "abc123", 0, "/input/video.mkv", "/output/video.mp4", "transcode", 1)
}

func TestNewJob(t *testing.T) {
	j := newTestJob()
	if j.Status != JobPending {
		t.Errorf("new job status = %v, want Pending", j.Status)
	}
	if j.TranscodeType != "transcode" {
		t.Errorf("transcode type = %q, want transcode", j.TranscodeType)
	}
}

func TestJob_Start(t *testing.T) {
	j := newTestJob()
	if err := j.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if j.Status != JobProcessing {
		t.Errorf("status = %v, want Processing", j.Status)
	}
	if j.StartedAt == 0 {
		t.Error("StartedAt should be set")
	}
}

func TestJob_Start_AlreadyRunning(t *testing.T) {
	j := newTestJob()
	j.Status = JobProcessing
	if err := j.Start(); err != ErrJobAlreadyRunning {
		t.Errorf("err = %v, want ErrJobAlreadyRunning", err)
	}
}

func TestJob_Complete(t *testing.T) {
	j := newTestJob()
	j.Status = JobProcessing
	if err := j.Complete("/output/video.mp4"); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if j.Status != JobCompleted {
		t.Errorf("status = %v, want Completed", j.Status)
	}
	if j.Progress != 100 {
		t.Errorf("progress = %d, want 100", j.Progress)
	}
	if j.CompletedAt == 0 {
		t.Error("CompletedAt should be set")
	}
}

func TestJob_Complete_NotProcessing(t *testing.T) {
	j := newTestJob()
	if err := j.Complete("/output"); err != ErrJobAlreadyRunning {
		t.Errorf("err = %v, want ErrJobAlreadyRunning", err)
	}
}

func TestJob_Fail(t *testing.T) {
	j := newTestJob()
	j.Status = JobProcessing
	if err := j.Fail("ffmpeg error"); err != nil {
		t.Fatalf("Fail returned error: %v", err)
	}
	if j.Status != JobFailed {
		t.Errorf("status = %v, want Failed", j.Status)
	}
	if j.ErrorMessage != "ffmpeg error" {
		t.Errorf("error message = %q", j.ErrorMessage)
	}
}

func TestJob_UpdateProgress(t *testing.T) {
	j := newTestJob()
	j.UpdateProgress(50)
	if j.Progress != 50 {
		t.Errorf("progress = %d, want 50", j.Progress)
	}
	j.UpdateProgress(-1)
	if j.Progress != 0 {
		t.Errorf("progress = %d, want 0 (clamped)", j.Progress)
	}
	j.UpdateProgress(150)
	if j.Progress != 100 {
		t.Errorf("progress = %d, want 100 (clamped)", j.Progress)
	}
}

func TestJob_CanRetry(t *testing.T) {
	j := newTestJob()
	if j.CanRetry() {
		t.Error("pending job should not be retryable")
	}
	j.Status = JobFailed
	if !j.CanRetry() {
		t.Error("failed job should be retryable")
	}
}

func TestJob_ResetForRetry(t *testing.T) {
	j := newTestJob()
	j.Status = JobFailed
	j.ErrorMessage = "error"
	j.Progress = 50
	if err := j.ResetForRetry(); err != nil {
		t.Fatalf("ResetForRetry failed: %v", err)
	}
	if j.Status != JobPending {
		t.Errorf("status = %v, want Pending", j.Status)
	}
	if j.Progress != 0 || j.ErrorMessage != "" {
		t.Error("fields should be cleared")
	}
}

func TestJob_ResetForRetry_NotFailed(t *testing.T) {
	j := newTestJob()
	if err := j.ResetForRetry(); err != ErrJobCannotRetry {
		t.Errorf("err = %v, want ErrJobCannotRetry", err)
	}
}
