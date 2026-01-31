// Package file provides file loading and saving utilities tests
// Author: Done-0
// Created: 2026-01-31
package file

import (
	"os"
	"path/filepath"
	"testing"
)

type sampleJSON struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestSaveAndLoadJSONFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "nested", "data.json")

	data := sampleJSON{Name: "demo", Count: 2}
	if err := SaveJSONFile(path, data); err != nil {
		t.Fatalf("SaveJSONFile() error = %v", err)
	}

	var loaded sampleJSON
	if err := LoadJSONFile(path, &loaded); err != nil {
		t.Fatalf("LoadJSONFile() error = %v", err)
	}

	if loaded != data {
		t.Fatalf("LoadJSONFile() = %+v, want %+v", loaded, data)
	}
}

func TestLoadJSONFile_FileNotFound(t *testing.T) {
	var target sampleJSON
	err := LoadJSONFile(filepath.Join(t.TempDir(), "missing.json"), &target)
	if err == nil {
		t.Fatal("LoadJSONFile() expected error for missing file, got nil")
	}
}

func TestLoadJSONFile_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var target sampleJSON
	err := LoadJSONFile(path, &target)
	if err == nil {
		t.Fatal("LoadJSONFile() expected error for invalid JSON, got nil")
	}
}

func TestSaveJSONFile_MarshalError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	ch := make(chan int)
	if err := SaveJSONFile(path, ch); err == nil {
		t.Fatal("SaveJSONFile() expected marshal error, got nil")
	}
}

func TestGetFileNameWithoutExt(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/tmp/video.mp4", "video"},
		{"relative/archive.tar.gz", "archive.tar"},
		{"noext", "noext"},
	}

	for _, tt := range tests {
		if got := GetFileNameWithoutExt(tt.path); got != tt.want {
			t.Fatalf("GetFileNameWithoutExt(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
