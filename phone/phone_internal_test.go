// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone

import (
	"reflect"
	"strings"
	"testing"
)

func TestPhoneDoesNotCacheLargeInputs(t *testing.T) {
	large := strings.Repeat("x", maxPhoneCacheKeyLength+1)
	before := len(phoneCache.data)

	m, err := New("US")
	if err != nil {
		t.Fatal(err)
	}

	result := m(reflect.ValueOf(large), nil)
	if result.Err == nil {
		t.Fatal("invalid number accepted")
	}

	if got := len(phoneCache.data); got != before {
		t.Fatalf("large input retained in phone cache: before=%d after=%d", before, got)
	}
}

func TestPhoneCacheSeparatesRegionAndNumber(t *testing.T) {
	t.Parallel()

	c := newCache()
	c.put(cacheKey{region: "A\x00B", number: "C"}, cacheEntry{value: "first"})
	c.put(cacheKey{region: "A", number: "B\x00C"}, cacheEntry{value: "second"})

	got, ok := c.get(cacheKey{region: "A\x00B", number: "C"})
	if !ok || got.value != "first" {
		t.Fatalf("distinct region and number pairs share a cache entry: %+v", got)
	}
}

func TestPhoneCacheEvictsOldestEntry(t *testing.T) {
	t.Parallel()

	c := newCache()

	c.cap = 2
	for _, number := range []string{"1", "2", "3"} {
		c.put(cacheKey{region: "US", number: number}, cacheEntry{value: number})
	}

	c.put(cacheKey{region: "US", number: "3"}, cacheEntry{value: "updated"})

	if _, ok := c.get(cacheKey{region: "US", number: "1"}); ok {
		t.Fatal("oldest entry was not evicted")
	}

	if got, _ := c.get(cacheKey{region: "US", number: "3"}); got.value != "updated" || len(c.keys) != 2 {
		t.Fatalf("update of an existing key changed the eviction order: %+v %v", got, c.keys)
	}
}
