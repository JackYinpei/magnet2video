// Package wire provides dependency injection configuration using Google Wire
// Author: Done-0
// Created: 2025-09-25
package wire

import (
	"github.com/google/wire"

	"magnet2video/internal/ai"
	"magnet2video/internal/cache"
	"magnet2video/internal/cloud"
	"magnet2video/internal/db"
	"magnet2video/internal/i18n"
	"magnet2video/internal/logger"
	"magnet2video/internal/queue"
	"magnet2video/internal/sse"
	"magnet2video/internal/tmdb"
	"magnet2video/internal/torrent"

	"magnet2video/internal/redis"
	"magnet2video/pkg/serve/controller"
	"magnet2video/pkg/serve/service"
	"magnet2video/pkg/serve/service/impl"
)

// InfrastructureProviders provides infrastructure layer dependencies
var InfrastructureProviders = wire.NewSet(
	ai.New,
	cache.New,
	cloud.New,
	db.New,
	logger.New,
	i18n.New,
	queue.NewProducer,
	redis.New,
	sse.New,
	tmdb.New,
	torrent.New,
)

// MapperProviders provides data access layer dependencies
var MapperProviders = wire.NewSet()

// ServiceProviders provides business logic layer dependencies
var ServiceProviders = wire.NewSet(
	impl.NewTestService,
	impl.NewTorrentService,
	wire.Bind(new(service.TorrentService), new(*impl.TorrentServiceImpl)),
	impl.NewUserService,
	impl.NewAdminService,
	impl.NewTranscodeService,
	wire.Bind(new(service.TranscodeService), new(*impl.TranscodeServiceImpl)),
)

// ControllerProviders provides controller layer dependencies
var ControllerProviders = wire.NewSet(
	controller.NewTestController,
	controller.NewTorrentController,
	controller.NewUserController,
	controller.NewAdminController,
)

// AllProviders combines all provider sets in dependency order
var AllProviders = wire.NewSet(
	InfrastructureProviders,
	MapperProviders,
	ServiceProviders,
	ControllerProviders,
)
