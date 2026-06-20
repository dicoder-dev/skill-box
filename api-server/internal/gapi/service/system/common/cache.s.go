package scommon

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var CacheInstance = CacheService{}

func InitCacheInstance(defaultExpiration, cleanupInterval time.Duration) {
	CacheInstance.Cache = cache.New(defaultExpiration, cleanupInterval)
}

type CacheService struct {
	Cache *cache.Cache
}

// 设置缓存项，可指定过期时间
func (s *CacheService) Set(key string, value interface{}, expiration time.Duration) {
	s.Cache.Set(key, value, expiration)
}

// 设置缓存项，使用默认过期时间
func (s *CacheService) SetDefault(key string, value interface{}) {
	s.Cache.SetDefault(key, value)
}

// 获取缓存项
func (s *CacheService) Get(key string) (interface{}, bool) {
	return s.Cache.Get(key)
}

// 删除缓存项
func (s *CacheService) Delete(key string) {
	s.Cache.Delete(key)
}

// 检查缓存项是否存在
func (s *CacheService) Exists(key string) bool {
	_, found := s.Cache.Get(key)
	return found
}

// 设置缓存项永不过期
func (s *CacheService) SetWithNoExpiration(key string, value interface{}) {
	s.Cache.Set(key, value, cache.NoExpiration)
}

// 获取缓存项数量
func (s *CacheService) ItemCount() int {
	return s.Cache.ItemCount()
}

// 清空所有缓存项
func (s *CacheService) Flush() {
	s.Cache.Flush()
}