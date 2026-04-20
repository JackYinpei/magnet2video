// Package cmd provides application startup and runtime entry point.
// Start() is the legacy all-mode entry kept for backward compatibility. New
// code should use RunMode() via main.go.
// Author: Done-0
// Created: 2025-09-25
package cmd

import (
	"log"

	"magnet2video/configs"
)

// Start boots the single-process (mode=all) application.
// Deprecated: prefer RunMode(mode) dispatched from main.go.
func Start() {
	RunMode(configs.ModeAll)
}

// RunMode boots the application in the given mode: "all", "server", "worker".
func RunMode(mode string) {
	if err := configs.New(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	cfg, err := configs.GetConfig()
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}

	// Allow flag-supplied mode to override config.
	if mode != "" {
		cfg.AppConfig.Mode = mode
	}
	if cfg.AppConfig.Mode == "" {
		cfg.AppConfig.Mode = configs.ModeAll
	}

	switch cfg.AppConfig.Mode {
	case configs.ModeServer:
		runServer(cfg)
	case configs.ModeWorker:
		runWorker(cfg)
	default:
		runAll(cfg)
	}
}
