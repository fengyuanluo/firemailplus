package cache

import (
	"sync"
	"time"
)

// Cache 通用缓存接口
type Cache interface {
	// Set 设置缓存项
	Set(key string, value interface{}, ttl time.Duration)
	
	// Get 获取缓存项
	Get(key string) (interface{}, bool)
	
	// Delete 删除缓存项
	Delete(key string)
	
	// Clear 清空所有缓存
	Clear()
	
	// Size 获取缓存项数量
	Size() int
	
	// Keys 获取所有键
	Keys() []string
}

// CacheItem 缓存项
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
	return time.Now().After(item.ExpiresAt)
}

// MemoryCache 基于内存的缓存实现
type MemoryCache struct {
	items sync.Map
	mutex sync.RWMutex
}

// NewMemoryCache 创建新的内存缓存
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{}
	
	// 启动清理协程
	go cache.startCleanup()
	
	return cache
}

// Set 设置缓存项
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	expiresAt := time.Now().Add(ttl)
	if ttl <= 0 {
		// 如果TTL为0或负数，设置为永不过期（100年后）
		expiresAt = time.Now().Add(100 * 365 * 24 * time.Hour)
	}
	
	item := &CacheItem{
		Value:     value,
		ExpiresAt: expiresAt,
	}
	
	c.items.Store(key, item)
}

// Get 获取缓存项
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	value, exists := c.items.Load(key)
	if !exists {
		return nil, false
	}
	
	item, ok := value.(*CacheItem)
	if !ok {
		c.items.Delete(key)
		return nil, false
	}
	
	if item.IsExpired() {
		c.items.Delete(key)
		return nil, false
	}
	
	return item.Value, true
}

// Delete 删除缓存项
func (c *MemoryCache) Delete(key string) {
	c.items.Delete(key)
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear() {
	c.items.Range(func(key, value interface{}) bool {
		c.items.Delete(key)
		return true
	})
}

// Size 获取缓存项数量
func (c *MemoryCache) Size() int {
	count := 0
	c.items.Range(func(key, value interface{}) bool {
		if item, ok := value.(*CacheItem); ok && !item.IsExpired() {
			count++
		}
		return true
	})
	return count
}

// Keys 获取所有有效的键
func (c *MemoryCache) Keys() []string {
	var keys []string
	c.items.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			if item, ok := value.(*CacheItem); ok && !item.IsExpired() {
				keys = append(keys, keyStr)
			}
		}
		return true
	})
	return keys
}

// startCleanup 启动定期清理过期项
func (c *MemoryCache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期项
func (c *MemoryCache) cleanup() {
	var expiredKeys []interface{}
	
	c.items.Range(func(key, value interface{}) bool {
		if item, ok := value.(*CacheItem); ok && item.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})
	
	for _, key := range expiredKeys {
		c.items.Delete(key)
	}
}

// CacheManager 缓存管理器
type CacheManager struct {
	emailListCache    Cache
	authCache         Cache
	providerCache     Cache
	folderStatusCache Cache
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{
		emailListCache:    NewMemoryCache(),
		authCache:         NewMemoryCache(),
		providerCache:     NewMemoryCache(),
		folderStatusCache: NewMemoryCache(),
	}
}

// EmailListCache 获取邮件列表缓存
func (cm *CacheManager) EmailListCache() Cache {
	return cm.emailListCache
}

// AuthCache 获取认证缓存
func (cm *CacheManager) AuthCache() Cache {
	return cm.authCache
}

// ProviderCache 获取提供商缓存
func (cm *CacheManager) ProviderCache() Cache {
	return cm.providerCache
}

// FolderStatusCache 获取文件夹状态缓存
func (cm *CacheManager) FolderStatusCache() Cache {
	return cm.folderStatusCache
}

// ClearAll 清空所有缓存
func (cm *CacheManager) ClearAll() {
	cm.emailListCache.Clear()
	cm.authCache.Clear()
	cm.providerCache.Clear()
	cm.folderStatusCache.Clear()
}

// GetStats 获取缓存统计信息
func (cm *CacheManager) GetStats() map[string]int {
	return map[string]int{
		"emailList":    cm.emailListCache.Size(),
		"auth":         cm.authCache.Size(),
		"provider":     cm.providerCache.Size(),
		"folderStatus": cm.folderStatusCache.Size(),
	}
}

// 全局缓存管理器实例
var GlobalCacheManager = NewCacheManager()
