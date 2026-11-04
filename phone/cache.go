// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone

import "sync"

const phoneCacheCapacity = 4096

// Inputs whose region and number together exceed this length are parsed
// normally but are not retained in the cache.
const maxPhoneCacheKeyLength = 256

// cacheKey separates region and number, so no pair can collide with another.
type cacheKey struct{ region, number string }

type cacheEntry struct {
	value string
	err   bool
}
type boundedCache struct {
	mu   sync.Mutex
	data map[cacheKey]cacheEntry
	keys []cacheKey
	cap  int
}

func newCache() *boundedCache {
	return &boundedCache{data: make(map[cacheKey]cacheEntry), cap: phoneCacheCapacity}
}
func (c *boundedCache) get(k cacheKey) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[k]
	return v, ok
}
func (c *boundedCache) put(k cacheKey, v cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.data[k]; !ok {
		if len(c.keys) >= c.cap {
			delete(c.data, c.keys[0])
			c.keys = c.keys[1:]
		}
		c.keys = append(c.keys, k)
	}
	c.data[k] = v
}

var phoneCache = newCache()
