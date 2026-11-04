// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizePhoneDoesNotCacheLargeInputs(t *testing.T) {
	large := strings.Repeat("x", maxPhoneCacheKeyLength+1)
	before := len(phoneCache.data)
	result := NormalizePhone(reflect.ValueOf(large), map[string]string{"region": "US"})
	if result.Error == nil {
		t.Fatal("invalid number accepted")
	}
	if got := len(phoneCache.data); got != before {
		t.Fatalf("large input retained in phone cache: before=%d after=%d", before, got)
	}
}

func TestPhoneCacheKeySeparatesRegionAndNumber(t *testing.T) {
	first := phoneCacheKey("A\x00B", "C")
	second := phoneCacheKey("A", "B\x00C")
	if first == second {
		t.Fatal("distinct region and number pairs have the same cache key")
	}
}
