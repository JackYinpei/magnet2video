// Package ai provides AI service functionality for multiple providers
// Author: Done-0
// Created: 2025-08-31
package ai

import (
	"magnet2video/configs"
	"magnet2video/internal/ai/internal"
	"magnet2video/internal/ai/internal/provider"
)

type (
	AIManager          = internal.Manager
	ChatRequest        = provider.ChatRequest
	ChatResponse       = provider.ChatResponse
	ChatStreamResponse = provider.ChatStreamResponse
	Choice             = provider.Choice
	Message            = provider.Message
	MessageDelta       = provider.MessageDelta
	Provider           = provider.Provider
	StreamChoice       = provider.StreamChoice
	Usage              = provider.Usage
)

// New creates a new AI manager instance
func New(config *configs.Config) (*AIManager, error) {
	return internal.New(config)
}
