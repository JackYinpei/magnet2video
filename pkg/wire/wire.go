//go:build wireinject

// Package wire provides Wire dependency injection definitions
// Author: Done-0
// Created: 2025-09-25
package wire

import (
	"github.com/google/wire"

	"magnet2video/configs"
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
)

// Container holds all application dependencies
type Container struct {
	Config *configs.Config

	// Infrastructure
	AIManager           *ai.AIManager
	CacheManager        cache.CacheManager
	CloudStorageManager cloud.CloudStorageManager
	DatabaseManager     db.DatabaseManager
	RedisManager        redis.RedisManager
	LoggerManager       logger.LoggerManager
	I18nManager         i18n.I18nManager
	SSEManager          sse.SSEManager
	TorrentManager      torrent.TorrentManager
	QueueProducer       queue.Producer
	TMDBClient          *tmdb.TMDBClient

	// Controllers
	TestController    *controller.TestController
	TorrentController *controller.TorrentController
	UserController    *controller.UserController
	AdminController   *controller.AdminController

	// Services
	TorrentService   service.TorrentService
	TranscodeService service.TranscodeService
}

// NewContainer initializes the complete application container using Wire
func NewContainer(config *configs.Config) (*Container, error) {
	panic(wire.Build(
		AllProviders,
		wire.Struct(new(Container), "*"),
	))
}
