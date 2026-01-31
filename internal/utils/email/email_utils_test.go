// Package email provides email utility tests
// Author: Done-0
// Created: 2026-01-31
package email

import "testing"

func TestNewRand_Range(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := NewRand()
		if code < 100000 || code > 999999 {
			t.Fatalf("NewRand() = %d, want 6-digit code", code)
		}
	}
}
