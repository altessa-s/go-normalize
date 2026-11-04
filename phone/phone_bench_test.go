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
	phone.Register(r)
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

func BenchmarkNormalizePhone(b *testing.B) {
	params := map[string]string{"region": "US"}
	b.Run("cached", func(b *testing.B) {
		v := reflect.ValueOf("(202) 555-0123")
		b.ReportAllocs()
		for b.Loop() {
			if out := phone.NormalizePhone(v, params); out.Error != nil {
				b.Fatal(out.Error)
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
			_ = phone.NormalizePhone(numbers[i%len(numbers)], params)
			i++
		}
	})
	b.Run("invalid", func(b *testing.B) {
		v := reflect.ValueOf("not a phone")
		b.ReportAllocs()
		for b.Loop() {
			if out := phone.NormalizePhone(v, params); out.Error == nil {
				b.Fatal("invalid number accepted")
			}
		}
	})
}

func BenchmarkNormalizePhoneParallel(b *testing.B) {
	params := map[string]string{"region": "US"}
	v := reflect.ValueOf("202-555-0123")
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if out := phone.NormalizePhone(v, params); out.Error != nil {
				b.Error(out.Error)
				return
			}
		}
	})
}
