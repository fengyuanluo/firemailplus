package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"firemail/internal/config"
)

// CacheItem 缓存项
type CacheItem struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
	return time.Now().After(item.ExpiresAt)
}

// CacheManager 缓存管理器接口
type CacheManager interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Exists(ctx context.Context, key string) bool
	GetStats() CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	TotalItems   int           `json:"total_items"`
	HitCount     int64         `json:"hit_count"`
	MissCount    int64         `json:"miss_count"`
	HitRate      float64       `json:"hit_rate"`
	MemoryUsage  int64         `json:"memory_usage"`
	LastCleanup  time.Time     `json:"last_cleanup"`
	CleanupCount int64         `json:"cleanup_count"`
}

// MemoryCacheManager 内存缓存管理器
type MemoryCacheManager struct {
	items       map[string]*CacheItem
	mutex       sync.RWMutex
	hitCount    int64
	missCount   int64
	lastCleanup time.Time
	cleanupCount int64
	
	// 配置
	maxItems     int
	cleanupInterval time.Duration
	defaultTTL   time.Duration
	
	// 清理定时器
	cleanupTimer *time.Timer
}

// NewMemoryCacheManager 创建内存缓存管理器
func NewMemoryCacheManager() CacheManager {
	manager := &MemoryCacheManager{
		items:           make(map[string]*CacheItem),
		maxItems:        10000, // 默认最大10000项
		cleanupInterval: 5 * time.Minute,
		defaultTTL:      30 * time.Minute,
		lastCleanup:     time.Now(),
	}
	
	// 根据配置调整参数
	if config.Env.IsTestMode() {
		manager.maxItems = 100
		manager.cleanupInterval = 1 * time.Minute
		manager.defaultTTL = 5 * time.Minute
	}
	
	// 启动定期清理
	manager.startCleanupTimer()
	
	return manager
}

// Set 设置缓存项
func (m *MemoryCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = m.defaultTTL
	}
	
	item := &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// 检查是否需要清理空间
	if len(m.items) >= m.maxItems {
		m.evictOldestItems(m.maxItems / 4) // 清理25%的空间
	}
	
	m.items[key] = item
	return nil
}

// Get 获取缓存项
func (m *MemoryCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	m.mutex.RLock()
	item, exists := m.items[key]
	m.mutex.RUnlock()
	
	if !exists {
		m.missCount++
		return fmt.Errorf("cache miss: key '%s' not found", key)
	}
	
	if item.IsExpired() {
		// 异步删除过期项
		go func() {
			m.mutex.Lock()
			delete(m.items, key)
			m.mutex.Unlock()
		}()
		
		m.missCount++
		return fmt.Errorf("cache miss: key '%s' expired", key)
	}
	
	m.hitCount++
	
	// 反序列化值
	if err := m.deserializeValue(item.Value, dest); err != nil {
		return fmt.Errorf("failed to deserialize cached value: %w", err)
	}
	
	return nil
}

// Delete 删除缓存项
func (m *MemoryCacheManager) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.items, key)
	return nil
}

// Clear 清空所有缓存
func (m *MemoryCacheManager) Clear(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.items = make(map[string]*CacheItem)
	m.hitCount = 0
	m.missCount = 0
	
	return nil
}

// Exists 检查键是否存在
func (m *MemoryCacheManager) Exists(ctx context.Context, key string) bool {
	m.mutex.RLock()
	item, exists := m.items[key]
	m.mutex.RUnlock()
	
	if !exists {
		return false
	}
	
	if item.IsExpired() {
		// 异步删除过期项
		go func() {
			m.mutex.Lock()
			delete(m.items, key)
			m.mutex.Unlock()
		}()
		return false
	}
	
	return true
}

// GetStats 获取缓存统计
func (m *MemoryCacheManager) GetStats() CacheStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	totalRequests := m.hitCount + m.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(m.hitCount) / float64(totalRequests)
	}
	
	// 估算内存使用量
	memoryUsage := int64(len(m.items) * 200) // 粗略估算每项200字节
	
	return CacheStats{
		TotalItems:   len(m.items),
		HitCount:     m.hitCount,
		MissCount:    m.missCount,
		HitRate:      hitRate,
		MemoryUsage:  memoryUsage,
		LastCleanup:  m.lastCleanup,
		CleanupCount: m.cleanupCount,
	}
}

// startCleanupTimer 启动清理定时器
func (m *MemoryCacheManager) startCleanupTimer() {
	m.cleanupTimer = time.AfterFunc(m.cleanupInterval, func() {
		m.cleanup()
		m.startCleanupTimer() // 重新启动定时器
	})
}

// cleanup 清理过期项
func (m *MemoryCacheManager) cleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	now := time.Now()
	expiredKeys := make([]string, 0)
	
	for key, item := range m.items {
		if item.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	for _, key := range expiredKeys {
		delete(m.items, key)
	}
	
	m.lastCleanup = now
	m.cleanupCount++
	
	if len(expiredKeys) > 0 {
		log.Printf("Cache cleanup: removed %d expired items", len(expiredKeys))
	}
}

// evictOldestItems 清理最旧的项目
func (m *MemoryCacheManager) evictOldestItems(count int) {
	if count <= 0 {
		return
	}
	
	// 收集所有项目并按创建时间排序
	type itemWithKey struct {
		key  string
		item *CacheItem
	}
	
	items := make([]itemWithKey, 0, len(m.items))
	for key, item := range m.items {
		items = append(items, itemWithKey{key: key, item: item})
	}
	
	// 简单排序：找到最旧的项目
	for i := 0; i < count && i < len(items); i++ {
		oldestIdx := i
		for j := i + 1; j < len(items); j++ {
			if items[j].item.CreatedAt.Before(items[oldestIdx].item.CreatedAt) {
				oldestIdx = j
			}
		}
		
		// 交换并删除
		if oldestIdx != i {
			items[i], items[oldestIdx] = items[oldestIdx], items[i]
		}
		delete(m.items, items[i].key)
	}
	
	log.Printf("Cache eviction: removed %d oldest items", count)
}

// deserializeValue 反序列化值
func (m *MemoryCacheManager) deserializeValue(value interface{}, dest interface{}) error {
	// 如果值已经是目标类型，直接赋值
	if reflect.TypeOf(value) == reflect.TypeOf(dest).Elem() {
		reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(value))
		return nil
	}
	
	// 否则通过JSON序列化/反序列化
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, dest)
}

// Stop 停止缓存管理器
func (m *MemoryCacheManager) Stop() {
	if m.cleanupTimer != nil {
		m.cleanupTimer.Stop()
	}
}
