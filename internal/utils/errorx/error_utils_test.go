// Package errorx provides error handling utilities tests
// Author: Done-0
// Created: 2026-01-31
package errorx

import (
	"errors"
	"testing"
)

func init() {
	// Register test error codes
	Register(90001, "test error")
	Register(90002, "error with param: {{.name}}")
	Register(90003, "multiple params: {{.a}} and {{.b}}")
}

func TestNew(t *testing.T) {
	err := New(90001)
	if err == nil {
		t.Fatal("New() should return error")
	}

	statusErr, ok := err.(StatusError)
	if !ok {
		t.Fatal("New() should return StatusError")
	}

	if statusErr.Code() != 90001 {
		t.Errorf("Code() = %d, want 90001", statusErr.Code())
	}
	if statusErr.Msg() != "test error" {
		t.Errorf("Msg() = %s, want 'test error'", statusErr.Msg())
	}
}

func TestNew_WithParam(t *testing.T) {
	err := New(90002, KV("name", "TestName"))
	if err == nil {
		t.Fatal("New() should return error")
	}

	statusErr := err.(StatusError)
	if statusErr.Msg() != "error with param: TestName" {
		t.Errorf("Msg() = %s, want 'error with param: TestName'", statusErr.Msg())
	}

	params := statusErr.Params()
	if params["name"] != "TestName" {
		t.Errorf("Params()[name] = %v, want TestName", params["name"])
	}
}

func TestNew_WithMultipleParams(t *testing.T) {
	err := New(90003, KV("a", "valueA"), KV("b", "valueB"))
	if err == nil {
		t.Fatal("New() should return error")
	}

	statusErr := err.(StatusError)
	if statusErr.Msg() != "multiple params: valueA and valueB" {
		t.Errorf("Msg() = %s, want 'multiple params: valueA and valueB'", statusErr.Msg())
	}
}

func TestNew_UnregisteredCode(t *testing.T) {
	err := New(99999) // Unregistered code
	if err == nil {
		t.Fatal("New() should return error even for unregistered code")
	}

	statusErr := err.(StatusError)
	if statusErr.Code() != 99999 {
		t.Errorf("Code() = %d, want 99999", statusErr.Code())
	}
	// Should use default message
	if statusErr.Msg() == "" {
		t.Error("Msg() should not be empty for unregistered code")
	}
}

func TestRegister(t *testing.T) {
	// Register a new error code
	Register(90100, "newly registered error")

	err := New(90100)
	statusErr := err.(StatusError)

	if statusErr.Msg() != "newly registered error" {
		t.Errorf("Msg() = %s, want 'newly registered error'", statusErr.Msg())
	}
}

func TestKV(t *testing.T) {
	opt := KV("key", "value")
	if opt == nil {
		t.Error("KV() should return non-nil Option")
	}
}

func TestStatusError_Extra(t *testing.T) {
	err := New(90001)
	statusErr := err.(StatusError)

	extra := statusErr.Extra()
	if extra == nil {
		t.Error("Extra() should not return nil")
	}
}

func TestStatusError_Error(t *testing.T) {
	err := New(90001)

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestStatusError_Interface(t *testing.T) {
	err := New(90001)

	// Should implement error interface
	var e error = err
	if e == nil {
		t.Error("StatusError should implement error interface")
	}

	// Should be unwrappable
	if !errors.Is(err, err) {
		t.Error("Error should be comparable with itself")
	}
}

func TestNew_MultipleCallsSameCode(t *testing.T) {
	err1 := New(90001)
	err2 := New(90001)

	// Should create separate instances
	if err1 == err2 {
		t.Error("New() should create new instances each time")
	}

	// But should have same code
	if err1.(StatusError).Code() != err2.(StatusError).Code() {
		t.Error("Same error code should produce same Code()")
	}
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(90001)
	}
}

func BenchmarkNew_WithParam(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New(90002, KV("name", "test"))
	}
}
