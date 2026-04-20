//go:build wireinject

// Package wire: provider sets used by wire-cli if/when it is re-introduced.
// Author: Done-0
// Created: 2025-09-25
//
// NOTE: The repository currently hand-maintains wire_gen.go rather than
// invoking `wire` during build. This file is kept under the wireinject tag so
// gopls excludes it from normal compilation; if you install the wire CLI,
// these provider sets mirror the wiring in wire_gen.go and can be used to
// regenerate it automatically.
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
	"magnet2video/internal/redis"
	"magnet2video/internal/sse"
	"magnet2video/internal/tmdb"
	"magnet2video/internal/torrent"

	"magnet2video/pkg/serve/controller"
	"magnet2video/pkg/serve/service"
	"magnet2video/pkg/serve/service/impl"
)

var InfrastructureProviders = wire.NewSet(
	ai.New, cache.New, cloud.New, db.New, logger.New, i18n.New,
	queue.NewProducer, redis.New, sse.New, tmdb.New, torrent.New,
)

var ServiceProviders = wire.NewSet(
	impl.NewTestService,
	impl.NewTorrentService,
	wire.Bind(new(service.TorrentService), new(*impl.TorrentServiceImpl)),
	impl.NewUserService,
	impl.NewAdminService,
	impl.NewTranscodeService,
	wire.Bind(new(service.TranscodeService), new(*impl.TranscodeServiceImpl)),
)

var ControllerProviders = wire.NewSet(
	controller.NewTestController,
	controller.NewTorrentController,
	controller.NewUserController,
	controller.NewAdminController,
	controller.NewWorkerController,
)

var AllProviders = wire.NewSet(
	InfrastructureProviders,
	ServiceProviders,
	ControllerProviders,
)
