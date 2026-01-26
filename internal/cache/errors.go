// Package cache 提供缓存层错误定义
// 创建者：Done-0
// 创建时间：2026-01-26
package cache

import "errors"

var (
	// ErrCacheMiss 表示键在缓存中不存在
	ErrCacheMiss = errors.New("cache miss")
)
