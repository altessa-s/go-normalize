// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/nyaruka/phonenumbers"

	"github.com/altessa-s/go-normalize/modifiers"
)

// ErrInvalidPhone identifies an invalid phone number.
var ErrInvalidPhone = errors.New("phone number is invalid")

// Register adds the phone modifier to registry. Registration is explicit and has no package side effect.
func Register(registry *modifiers.Registry) { registry.Register("phone", NormalizePhone) }

// NormalizePhone converts a valid phone number to E.164 form. Region defaults to RU.
func NormalizePhone(v reflect.Value, params map[string]string) modifiers.ModifierResult {
	if !v.IsValid() {
		return modifiers.ModifierResult{Value: v}
	}
	var s string
	ptr := false
	switch v.Kind() {
	case reflect.String:
		s = v.String()
	case reflect.Pointer:
		if v.IsNil() || v.Type().Elem().Kind() != reflect.String {
			return modifiers.ModifierResult{Value: v}
		}
		s = v.Elem().String()
		ptr = true
	default:
		return modifiers.ModifierResult{Value: v}
	}
	if s == "" {
		return modifiers.ModifierResult{Value: v}
	}
	region := strings.ToUpper(params["region"])
	if region == "" {
		region = "RU"
	}
	keyLength := len(region) + len(s) + len(strconv.Itoa(len(region))) + 1
	cacheable := keyLength <= maxPhoneCacheKeyLength
	var key string
	if cacheable {
		key = phoneCacheKey(region, s)
		if got, ok := phoneCache.get(key); ok {
			if got.err {
				return fail(v)
			}
			return set(v, got.value, ptr)
		}
	}
	cleaned := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "", "_", "").Replace(s)
	n, err := phonenumbers.Parse(cleaned, region)
	if err != nil || !phonenumbers.IsValidNumber(n) {
		if cacheable {
			phoneCache.put(key, cacheEntry{err: true})
		}
		return fail(v)
	}
	out := phonenumbers.Format(n, phonenumbers.E164)
	if cacheable {
		phoneCache.put(key, cacheEntry{value: out})
	}
	return set(v, out, ptr)
}

func phoneCacheKey(region, number string) string {
	// Length-prefix the region so arbitrary parameter strings cannot collide
	// across the separator with a different region/number pair.
	return strconv.Itoa(len(region)) + ":" + region + number
}
func fail(v reflect.Value) modifiers.ModifierResult {
	return modifiers.ModifierResult{Value: v, Error: &modifiers.ModifierError{ModifierName: "phone", Cause: ErrInvalidPhone}}
}
func set(v reflect.Value, s string, ptr bool) modifiers.ModifierResult {
	if ptr {
		x := reflect.New(v.Type().Elem())
		x.Elem().SetString(s)
		if v.CanSet() {
			v.Set(x)
			return modifiers.ModifierResult{Value: v}
		}
		return modifiers.ModifierResult{Value: x}
	}
	if s == v.String() {
		return modifiers.ModifierResult{Value: v}
	}
	if v.CanSet() {
		v.SetString(s)
		return modifiers.ModifierResult{Value: v}
	}
	x := reflect.New(v.Type()).Elem()
	x.SetString(s)
	return modifiers.ModifierResult{Value: x}
}

const phoneCacheCapacity = 4096

// Large inputs are parsed normally but are not retained in the cache.
const maxPhoneCacheKeyLength = 256

type cacheEntry struct {
	value string
	err   bool
}
type boundedCache struct {
	mu   sync.Mutex
	data map[string]cacheEntry
	keys []string
	cap  int
}

func newCache() *boundedCache {
	return &boundedCache{data: make(map[string]cacheEntry), cap: phoneCacheCapacity}
}
func (c *boundedCache) get(k string) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[k]
	return v, ok
}
func (c *boundedCache) put(k string, v cacheEntry) {
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
