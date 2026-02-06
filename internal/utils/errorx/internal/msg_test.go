// Package internal provides tests for message wrapping
// Author: Done-0
// Created: 2026-02-04
package internal

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapf(t *testing.T) {
	if wrapf(nil, "ignore %s", "x") != nil {
		t.Fatalf("wrapf(nil) should return nil")
	}

	base := errors.New("root")
	wrapped := wrapf(base, "hello %s", "world")
	if wrapped == nil {
		t.Fatalf("wrapf() should return error")
	}

	msg := wrapped.Error()
	if !strings.Contains(msg, "hello world") {
		t.Fatalf("wrapf() message = %q, want to contain formatted msg", msg)
	}
	if !strings.Contains(msg, "cause=root") {
		t.Fatalf("wrapf() message = %q, want to contain cause", msg)
	}
}

func TestWrapfAddsStack(t *testing.T) {
	base := errors.New("root")
	wrapped := Wrapf(base, "oops")
	if wrapped == nil {
		t.Fatalf("Wrapf() should return error")
	}
	if _, ok := wrapped.(StackTracer); !ok {
		t.Fatalf("Wrapf() should add stack trace")
	}
}
