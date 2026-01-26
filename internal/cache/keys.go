// Package cache 提供缓存键生成工具
// 创建者：Done-0
// 创建时间：2026-01-26
package cache

import (
	"fmt"
	"time"
)

// 缓存键前缀，用于命名空间隔离不同类型的缓存
const (
	PrefixTorrent = "torrent:" // Torrent 相关缓存键前缀
	PrefixUser    = "user:"    // 用户相关缓存键前缀
)

// 缓存 TTL 配置
const (
	// TTLTorrentList 用户 torrent 列表缓存的 TTL
	// 设置较短的 TTL 因为下载进度变化频繁
	TTLTorrentList = 30 * time.Second

	// TTLTorrentDetail torrent 详情缓存的 TTL
	TTLTorrentDetail = 2 * time.Minute

	// TTLPublicList 公共 torrent 列表缓存的 TTL
	// 设置稍长因为所有用户共享
	TTLPublicList = 1 * time.Minute
)

// TorrentListKey 生成用户 torrent 列表的缓存键
// 格式: torrent:list:user:{userID}
func TorrentListKey(userID int64) string {
	return fmt.Sprintf("%slist:user:%d", PrefixTorrent, userID)
}

// PublicTorrentListKey 生成公共 torrent 列表的缓存键
// 格式: torrent:list:public
func PublicTorrentListKey() string {
	return fmt.Sprintf("%slist:public", PrefixTorrent)
}

// TorrentDetailKey 生成 torrent 详情的缓存键
// 格式: torrent:detail:{infoHash}
func TorrentDetailKey(infoHash string) string {
	return fmt.Sprintf("%sdetail:%s", PrefixTorrent, infoHash)
}

// TorrentProgressKey 生成 torrent 进度的缓存键
// 格式: torrent:progress:{infoHash}
func TorrentProgressKey(infoHash string) string {
	return fmt.Sprintf("%sprogress:%s", PrefixTorrent, infoHash)
}

// InvalidateUserTorrentsPattern 返回用于失效所有用户 torrent 缓存的模式
// 格式: torrent:list:user:*
func InvalidateUserTorrentsPattern() string {
	return fmt.Sprintf("%slist:user:*", PrefixTorrent)
}

// InvalidateTorrentCachesPattern 返回用于失效所有 torrent 缓存的模式
// 格式: torrent:*
func InvalidateTorrentCachesPattern() string {
	return fmt.Sprintf("%s*", PrefixTorrent)
}
