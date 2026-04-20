//go:build wireinject

// Package wire provides Wire dependency injection definitions
// Author: Done-0
// Created: 2025-09-25
//
// The actual wiring lives in wire_gen.go (hand-maintained). These stubs exist
// so the wire CLI can optionally regenerate wire_gen.go in the future.
package wire

import (
	"github.com/google/wire"

	"magnet2video/configs"
)

// NewContainer initializes the single-process (mode=all) container.
func NewContainer(config *configs.Config) (*Container, error) {
	panic(wire.Build(AllProviders, wire.Struct(new(Container), "*")))
}

// NewServerContainer initializes the server-only container.
func NewServerContainer(config *configs.Config) (*Container, error) {
	panic(wire.Build(AllProviders, wire.Struct(new(Container), "*")))
}

// NewWorkerContainer initializes the worker-only container.
func NewWorkerContainer(config *configs.Config) (*Container, error) {
	panic(wire.Build(AllProviders, wire.Struct(new(Container), "*")))
}
