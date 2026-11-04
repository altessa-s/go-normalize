// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/phone"
)

func BenchmarkPhone(b *testing.B) {
	r := modifiers.NewRegistry()
	if err := phone.Register(r, "US"); err != nil {
		b.Fatal(err)
	}

	type contact struct {
		Phone string `normalize:"phone(region=US)"`
	}

	b.ReportAllocs()

	for b.Loop() {
		v := contact{Phone: "2025550123"}
		if err := normalize.NormalizeWithRegistry(&v, r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPhoneModifier(b *testing.B) {
	params := map[string]string{"region": "US"}
	m := newModifier(b, "RU")

	b.Run("cached", func(b *testing.B) {
		v := reflect.ValueOf("(202) 555-0123")

		b.ReportAllocs()

		for b.Loop() {
			if out := m(v, params); out.Err != nil {
				b.Fatal(out.Err)
			}
		}
	})
	b.Run("uncached", func(b *testing.B) {
		// Twice the cache capacity, walked in order, so FIFO eviction misses every lookup.
		numbers := make([]reflect.Value, 8192)
		for i := range numbers {
			numbers[i] = reflect.ValueOf(fmt.Sprintf("(202) 555-%04d", i))
		}

		b.ReportAllocs()

		i := 0
		for b.Loop() {
			_ = m(numbers[i%len(numbers)], params)
			i++
		}
	})
	b.Run("invalid", func(b *testing.B) {
		v := reflect.ValueOf("not a phone")

		b.ReportAllocs()

		for b.Loop() {
			if out := m(v, params); out.Err == nil {
				b.Fatal("invalid number accepted")
			}
		}
	})
}

func BenchmarkPhoneModifierParallel(b *testing.B) {
	params := map[string]string{"region": "US"}
	m := newModifier(b, "RU")
	v := reflect.ValueOf("202-555-0123")

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if out := m(v, params); out.Err != nil {
				b.Error(out.Err)
				return
			}
		}
	})
}
