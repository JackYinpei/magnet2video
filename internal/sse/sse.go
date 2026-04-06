// Package sse provides Server-Sent Events functionality
// Author: Done-0
// Created: 2025-08-31
package sse

import (
	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"

	"magnet2video/configs"
	"magnet2video/internal/sse/internal"
)

// SSEManager defines SSE operations
type SSEManager interface {
	StreamToClient(c *gin.Context, events <-chan *Event) error
}

// Event represents a Server-Sent Event
type (
	Event = sse.Event
)

// New creates SSE manager
func New(config *configs.Config) SSEManager {
	return internal.NewManager(config)
}
