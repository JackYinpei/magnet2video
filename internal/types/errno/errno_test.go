// Package errno provides tests for error code registrations
// Author: Done-0
// Created: 2026-02-04
package errno

import (
	"testing"

	"magnet2video/internal/utils/errorx"
)

func TestErrorMessages(t *testing.T) {
	cases := []struct {
		name string
		code int32
		opts []errorx.Option
		want string
	}{
		{
			name: "system internal",
			code: ErrInternalServer,
			opts: []errorx.Option{errorx.KV("msg", "boom")},
			want: "internal server error: boom",
		},
		{
			name: "system forbidden",
			code: ErrForbidden,
			opts: []errorx.Option{errorx.KV("resource", "admin")},
			want: "permission denied: admin",
		},
		{
			name: "torrent file not streamable",
			code: ErrFileNotStreamable,
			opts: []errorx.Option{errorx.KV("path", "/tmp/video.mkv")},
			want: "file is not streamable: /tmp/video.mkv",
		},
		{
			name: "transcode failed",
			code: ErrTranscodeFailed,
			opts: []errorx.Option{errorx.KV("error", "ffmpeg")},
			want: "transcoding failed: ffmpeg",
		},
		{
			name: "cloud upload failed",
			code: ErrCloudUploadFailed,
			opts: []errorx.Option{errorx.KV("msg", "network")},
			want: "cloud upload failed: network",
		},
		{
			name: "admin resources",
			code: ErrUserHasResources,
			opts: []errorx.Option{errorx.KV("count", "5")},
			want: "user has 5 resources, please delete them first",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := errorx.New(tc.code, tc.opts...)
			statusErr, ok := err.(errorx.StatusError)
			if !ok {
				t.Fatalf("errorx.New() type = %T, want StatusError", err)
			}
			if statusErr.Msg() != tc.want {
				t.Fatalf("Msg() = %q, want %q", statusErr.Msg(), tc.want)
			}
		})
	}
}

func TestErrorMessages_NoParams(t *testing.T) {
	err := errorx.New(ErrAdminRequired)
	statusErr, ok := err.(errorx.StatusError)
	if !ok {
		t.Fatalf("errorx.New() type = %T, want StatusError", err)
	}
	if statusErr.Msg() != "admin permission required" {
		t.Fatalf("Msg() = %q, want %q", statusErr.Msg(), "admin permission required")
	}
}
