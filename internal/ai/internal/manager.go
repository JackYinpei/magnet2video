// Package internal provides AI service internal implementation
// Author: Done-0
// Created: 2025-08-31
package internal

import (
	"magnet2video/configs"
	"magnet2video/internal/ai/internal/prompter"
	"magnet2video/internal/ai/internal/provider"
)

type Manager struct {
	provider.Provider
	prompter.Prompter
}

// New creates a new AI provider manager with dynamic prompt loading
func New(config *configs.Config) (*Manager, error) {
	return &Manager{
		Provider: provider.New(),
		Prompter: prompter.New(),
	}, nil
}
