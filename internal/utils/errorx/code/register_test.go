// Package code provides tests for error code registration adapter
// Author: Done-0
// Created: 2026-02-04
package code

import (
	"testing"

	"magnet2video/internal/utils/errorx"
)

func TestRegister(t *testing.T) {
	const code = 91001
	Register(code, "registered: {{.name}}")

	err := errorx.New(code, errorx.KV("name", "Alice"))
	statusErr, ok := err.(errorx.StatusError)
	if !ok {
		t.Fatalf("errorx.New() type = %T, want StatusError", err)
	}
	if statusErr.Msg() != "registered: Alice" {
		t.Fatalf("Msg() = %q, want %q", statusErr.Msg(), "registered: Alice")
	}
}
