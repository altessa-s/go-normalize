// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

func BenchmarkParse(b *testing.B) {
	cases := []struct {
		name   string
		clause string
	}{
		{name: "plain", clause: "trim"},
		{name: "params", clause: "phone(region=US)"},
		{name: "uncached", clause: "phone(region=" + strings.Repeat("x", 1024) + ")"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = modifiers.Parse(tc.clause)
			}
		})
	}
}

func BenchmarkBuiltins(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "trim", input: "  value  "},
		{name: "lowercase", input: "VALUE"},
		{name: "uppercase", input: "value"},
		{name: "remove_bad_symbols", input: "va\u202elue\x00"},
	}
	r := modifiers.DefaultRegistry()
	for _, tc := range cases {
		m, ok := r.Get(tc.name)
		if !ok {
			b.Fatalf("missing built-in %s", tc.name)
		}
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			var s string
			v := reflect.ValueOf(&s).Elem()
			for b.Loop() {
				s = tc.input
				if out := m(v, nil); out.Error != nil {
					b.Fatal(out.Error)
				}
			}
		})
	}
}

func BenchmarkNilOnEmpty(b *testing.B) {
	b.ReportAllocs()
	var p *string
	v := reflect.ValueOf(&p).Elem()
	for b.Loop() {
		p = new("")
		modifiers.NilOnEmpty(v, nil)
	}
}

func BenchmarkRemoveEmptyElements(b *testing.B) {
	template := []string{"a", " ", "b", "", "c", "d", " ", "e"}
	b.ReportAllocs()
	var s []string
	v := reflect.ValueOf(&s).Elem()
	for b.Loop() {
		s = template
		modifiers.RemoveEmptyElements(v, nil)
	}
}

func BenchmarkRegistryGetParallel(b *testing.B) {
	r := modifiers.DefaultRegistry()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, ok := r.Get("trim"); !ok {
				b.Error("missing trim")
				return
			}
		}
	})
}
