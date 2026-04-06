// Package internal provides tests for stack handling
// Author: Done-0
// Created: 2026-02-04
package internal

import (
	"errors"
	"strings"
	"testing"
)

func TestStack(t *testing.T) {
	trace := stack()
	if trace == "" {
		t.Fatalf("stack() should return non-empty stack")
	}
	if !strings.Contains(trace, "TestStack") {
		t.Fatalf("stack() should include caller function name, got: %q", trace)
	}
}

func TestTrimPathPrefix(t *testing.T) {
	cases := map[string]string{
		"magnet2video/internal/utils/errorx/internal.stack": "stack",
		"main.main": "main",
		"stack": "stack",
	}

	for input, want := range cases {
		if got := trimPathPrefix(input); got != want {
			t.Fatalf("trimPathPrefix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWithStackTraceIfNotExists(t *testing.T) {
	if withStackTraceIfNotExists(nil) != nil {
		t.Fatalf("withStackTraceIfNotExists(nil) should return nil")
	}

	base := errors.New("boom")
	wrapped := withStackTraceIfNotExists(base)
	if wrapped == nil || wrapped == base {
		t.Fatalf("withStackTraceIfNotExists() should wrap non-stack errors")
	}
	if _, ok := wrapped.(StackTracer); !ok {
		t.Fatalf("withStackTraceIfNotExists() result should implement StackTracer")
	}

	stacked := &withStack{cause: base, stack: "trace"}
	again := withStackTraceIfNotExists(stacked)
	if again != stacked {
		t.Fatalf("withStackTraceIfNotExists() should not wrap stack errors")
	}
}
