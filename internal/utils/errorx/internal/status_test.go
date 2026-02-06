// Package internal provides tests for status error handling
// Author: Done-0
// Created: 2026-02-04
package internal

import (
	"errors"
	"strings"
	"testing"
)

func resetCodeDefinitions() {
	CodeDefinitions = make(map[int32]*CodeDefinition)
}

func TestParamAndExtra(t *testing.T) {
	resetCodeDefinitions()
	Register(100, "hello {{.name}}")

	ws := &withStatus{status: getStatusByCode(100)}
	Param("name", "Bob")(ws)
	Extra("trace", "abc")(ws)

	if ws.status.message != "hello Bob" {
		t.Fatalf("Param() message = %q, want %q", ws.status.message, "hello Bob")
	}
	if ws.status.params["name"] != "Bob" {
		t.Fatalf("Param() params[name] = %v, want %q", ws.status.params["name"], "Bob")
	}
	if ws.status.extra["trace"] != "abc" {
		t.Fatalf("Extra() extra[trace] = %v, want %q", ws.status.extra["trace"], "abc")
	}

	// Should be no-op on nil receiver
	Param("x", "y")(nil)
	Extra("x", "y")(nil)
}

func TestNewByCode_DefaultMessage(t *testing.T) {
	resetCodeDefinitions()

	err := NewByCode(99999)
	ws, ok := err.(*withStatus)
	if !ok {
		t.Fatalf("NewByCode() type = %T, want *withStatus", err)
	}
	if ws.status.message != DefaultErrorMsg {
		t.Fatalf("NewByCode() message = %q, want %q", ws.status.message, DefaultErrorMsg)
	}
	if ws.status.code != 99999 {
		t.Fatalf("NewByCode() code = %d, want %d", ws.status.code, 99999)
	}
}

func TestWrapByCode(t *testing.T) {
	resetCodeDefinitions()
	Register(200, "boom")

	base := errors.New("root")
	err := WrapByCode(base, 200)
	ws := err.(*withStatus)

	if ws.cause != base {
		t.Fatalf("WrapByCode() cause = %v, want %v", ws.cause, base)
	}
	if ws.stack == "" {
		t.Fatalf("WrapByCode() should include stack for non-stacked errors")
	}
}

func TestWrapByCode_Nil(t *testing.T) {
	resetCodeDefinitions()
	if err := WrapByCode(nil, 200); err != nil {
		t.Fatalf("WrapByCode(nil) = %v, want nil", err)
	}
}

func TestWrapByCode_ExistingStack(t *testing.T) {
	resetCodeDefinitions()
	Register(300, "with stack")

	base := errors.New("root")
	stacked := withStackTraceIfNotExists(base)

	err := WrapByCode(stacked, 300)
	ws := err.(*withStatus)
	if ws.stack != "" {
		t.Fatalf("WrapByCode() stack = %q, want empty because stack already exists", ws.stack)
	}
	if ws.cause != stacked {
		t.Fatalf("WrapByCode() cause = %v, want %v", ws.cause, stacked)
	}
}

func TestWithStatus_IsAndAs(t *testing.T) {
	resetCodeDefinitions()
	Register(400, "first")
	Register(401, "second")

	err1 := NewByCode(400)
	err2 := NewByCode(400)
	err3 := NewByCode(401)

	if !errors.Is(err1, err2) {
		t.Fatalf("errors.Is should match same code")
	}
	if errors.Is(err1, err3) {
		t.Fatalf("errors.Is should not match different code")
	}
	if errors.Is(err1, errors.New("other")) {
		t.Fatalf("errors.Is should not match unrelated error")
	}

	var se *statusError
	if !errors.As(err1, &se) {
		t.Fatalf("errors.As should extract *statusError")
	}
	if se.code != 400 {
		t.Fatalf("errors.As status code = %d, want %d", se.code, 400)
	}
}

func TestWithStatus_ErrorString(t *testing.T) {
	ws := &withStatus{
		status: &statusError{code: 7, message: "message"},
		cause:  errors.New("cause"),
		stack:  "stack",
	}

	msg := ws.Error()
	if !strings.Contains(msg, "code=7 message=message") {
		t.Fatalf("Error() missing status info: %q", msg)
	}
	if !strings.Contains(msg, "cause=cause") {
		t.Fatalf("Error() missing cause: %q", msg)
	}
	if !strings.Contains(msg, "stack=stack") {
		t.Fatalf("Error() missing stack: %q", msg)
	}
}
