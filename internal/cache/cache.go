// Package cache 提供缓存层抽象和工具
// 创建者：Done-0
// 创建时间：2026-01-26
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"magnet2video/internal/logger"
	redisManager "magnet2video/internal/redis"
)

// CacheManager，缓存操作接口，定义了所有缓存相关的方法
type CacheManager interface {
	// Get 从缓存中获取值
	Get(ctx context.Context, key string, dest any) error
	// Set 将值存储到缓存中并设置 TTL
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Delete 从缓存中删除指定的键
	Delete(ctx context.Context, keys ...string) error
	// DeleteByPattern 删除所有匹配指定模式的键
	DeleteByPattern(ctx context.Context, pattern string) error
	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// GetOrSet 从缓存获取值，若不存在则使用 loader 函数加载并存储
	GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, loader func() (any, error)) error
}

// Manager，缓存管理器实现，使用 Redis 作为缓存后端
type Manager struct {
	redis  redisManager.RedisManager // Redis 连接管理器
	logger logger.LoggerManager      // 日志管理器
}

// New 创建新的缓存管理器实例
func New(redis redisManager.RedisManager, logger logger.LoggerManager) CacheManager {
	return &Manager{
		redis:  redis,
		logger: logger,
	}
}

// Get 从缓存中获取值并反序列化到目标对象
func (m *Manager) Get(ctx context.Context, key string, dest any) error {
	data, err := m.redis.Client().Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("cache get failed: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("cache unmarshal failed: %w", err)
	}

	return nil
}

// Set 将值序列化后存储到缓存中
func (m *Manager) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}

	if err := m.redis.Client().Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}

	return nil
}

// Delete 从缓存中删除一个或多个键
func (m *Manager) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	if err := m.redis.Client().Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}

	return nil
}

// DeleteByPattern 使用 SCAN 命令删除所有匹配模式的键
func (m *Manager) DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var allKeys []string

	for {
		keys, nextCursor, err := m.redis.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan failed: %w", err)
		}

		allKeys = append(allKeys, keys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	if len(allKeys) > 0 {
		if err := m.redis.Client().Del(ctx, allKeys...).Err(); err != nil {
			return fmt.Errorf("cache delete by pattern failed: %w", err)
		}
		m.logger.Logger().Debugf("已删除 %d 个匹配模式 %s 的缓存键", len(allKeys), pattern)
	}

	return nil
}

// Exists 检查指定键是否存在于缓存中
func (m *Manager) Exists(ctx context.Context, key string) (bool, error) {
	count, err := m.redis.Client().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists check failed: %w", err)
	}
	return count > 0, nil
}

// GetOrSet 实现 Cache-Aside 模式，优先从缓存获取，未命中时从源加载
func (m *Manager) GetOrSet(ctx context.Context, key string, dest any, ttl time.Duration, loader func() (any, error)) error {
	// 尝试从缓存获取
	err := m.Get(ctx, key, dest)
	if err == nil {
		m.logger.Logger().Debugf("缓存命中: %s", key)
		return nil
	}

	if err != ErrCacheMiss {
		// 记录非预期错误但继续从源加载
		m.logger.Logger().Warnf("缓存获取错误，回退到 loader: %v", err)
	}

	m.logger.Logger().Debugf("缓存未命中: %s", key)

	// 从源加载数据
	value, err := loader()
	if err != nil {
		return err
	}

	// 异步存储到缓存，不阻塞响应
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if setErr := m.Set(cacheCtx, key, value, ttl); setErr != nil {
			m.logger.Logger().Warnf("缓存存储失败: %v", setErr)
		}
	}()

	// 将值复制到目标对象
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
