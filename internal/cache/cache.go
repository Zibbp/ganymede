package cache

import (
	"time"

	goCache "github.com/patrickmn/go-cache"
)

var cache *Store

type Store struct {
	cache *goCache.Cache
}

func init() {
	c := goCache.New(5*time.Minute, 1*time.Minute)
	cache = &Store{cache: c}
}

func Cache() *Store {
	return cache
}

func (c Store) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c Store) Set(key string, value interface{}, expiration time.Duration) error {
	c.cache.Set(key, value, expiration)

	return nil
}
