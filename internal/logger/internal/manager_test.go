// Package internal provides tests for logger manager
// Author: Done-0
// Created: 2026-02-04
package internal

import (
	"testing"

	"github.com/sirupsen/logrus"

	"magnet2video/configs"
)

func TestManagerInitializeAndClose(t *testing.T) {
	dir := t.TempDir()
	config := &configs.Config{
		LogConfig: configs.LogConfig{
			LogFilePath: dir,
			LogFileName: "app.log",
			LogLevel:    "info",
			LogMaxAge:   1,
		},
	}

	m, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if m.logger == nil {
		t.Fatalf("Initialize() should set logger")
	}
	if m.Logger() == nil {
		t.Fatalf("Logger() should return logger")
	}

	if err := m.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if m.logger != nil {
		t.Fatalf("Close() should clear logger")
	}
	if m.logFile != nil {
		t.Fatalf("Close() should clear log file handle")
	}
}

func TestManagerInvalidLogLevel(t *testing.T) {
	dir := t.TempDir()
	config := &configs.Config{
		LogConfig: configs.LogConfig{
			LogFilePath: dir,
			LogFileName: "app.log",
			LogLevel:    "not-a-level",
			LogMaxAge:   1,
		},
	}

	m, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if m.logger.Level != logrus.InfoLevel {
		t.Fatalf("Logger level = %v, want info", m.logger.Level)
	}

	if err := m.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestManagerCloseWithoutLogger(t *testing.T) {
	m := &Manager{}
	if err := m.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}
