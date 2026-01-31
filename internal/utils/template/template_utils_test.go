// Package template provides template variable substitution utilities tests
// Author: Done-0
// Created: 2026-01-31
package template

import (
	"strings"
	"testing"
	"time"
)

func TestReplace_EmptyText(t *testing.T) {
	result, err := Replace("", map[string]any{"Name": "World"})
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if result != "" {
		t.Fatalf("Replace() result = %q, want empty string", result)
	}
}

func TestReplace_Basic(t *testing.T) {
	unixTime := int64(1700000000)
	vars := map[string]any{
		"Name": "Done",
		"Unix": unixTime,
	}

	text := "Hello {{.Name}}, sum={{ add 2 3 }}, time={{ unixToTime .Unix }}"
	result, err := Replace(text, vars)
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	expectedTime := time.Unix(unixTime, 0).Format("2006年01月02日 15时04分")
	if !strings.Contains(result, "Hello Done") {
		t.Fatalf("Replace() result missing greeting: %q", result)
	}
	if !strings.Contains(result, "sum=5") {
		t.Fatalf("Replace() result missing sum: %q", result)
	}
	if !strings.Contains(result, expectedTime) {
		t.Fatalf("Replace() result missing time %q: %q", expectedTime, result)
	}
}

func TestReplace_ParseError(t *testing.T) {
	_, err := Replace("{{ .Name ", map[string]any{"Name": "Done"})
	if err == nil {
		t.Fatal("Replace() expected parse error, got nil")
	}
}

func TestReplace_ExecuteError(t *testing.T) {
	_, err := Replace("{{ call .Fn }}", map[string]any{"Fn": 123})
	if err == nil {
		t.Fatal("Replace() expected execute error, got nil")
	}
}
