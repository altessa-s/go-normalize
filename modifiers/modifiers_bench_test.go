// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
		{name: "collapse_spaces", input: "a  b\t\n c"},
		{name: "digits_only", input: "+7 (916) 123-45-67"},
		{name: "email", input: " John.Doe@Example.COM "},
		{name: "url", input: "HTTPS://Example.COM:443/path?q=1"},
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

func BenchmarkApplyModifier(b *testing.B) {
	clamp := func(n int) (int, bool) { return min(n, 150), n > 150 }

	var n int

	v := reflect.ValueOf(&n).Elem()

	b.ReportAllocs()

	for b.Loop() {
		n = 200

		modifiers.ApplyModifier(v, clamp)
	}
}

func BenchmarkTruncate(b *testing.B) {
	params := map[string]string{"max": "16"}

	b.ReportAllocs()

	var s string

	v := reflect.ValueOf(&s).Elem()
	for b.Loop() {
		s = "\u043d\u043e\u0440\u043c\u0430\u043b\u0438\u0437\u0430\u0446\u0438\u044f \u0441\u0442\u0440\u043e\u043a\u0438"

		if out := modifiers.Truncate(v, params); out.Error != nil {
			b.Fatal(out.Error)
		}
	}
}

func BenchmarkDedupe(b *testing.B) {
	template := []string{"go", "rust", "go", "zig", "rust", "c", "go", "odin"}

	b.ReportAllocs()

	var s []string

	v := reflect.ValueOf(&s).Elem()
	for b.Loop() {
		s = template

		modifiers.Dedupe(v, nil)
	}
}

func BenchmarkNilOnZero(b *testing.B) {
	b.ReportAllocs()

	var p *int

	v := reflect.ValueOf(&p).Elem()
	for b.Loop() {
		p = new(0)

		modifiers.NilOnZero(v, nil)
	}
}

func BenchmarkTruncateTime(b *testing.B) {
	params := map[string]string{"unit": "us"}
	at := time.Date(2026, 3, 4, 5, 6, 7, 891234567, time.UTC)

	b.ReportAllocs()

	var ts time.Time

	v := reflect.ValueOf(&ts).Elem()
	for b.Loop() {
		ts = at

		if out := modifiers.TruncateTime(v, params); out.Error != nil {
			b.Fatal(out.Error)
		}
	}
}

func BenchmarkClamp(b *testing.B) {
	params := map[string]string{"min": "1", "max": "100"}

	b.ReportAllocs()

	var n int

	v := reflect.ValueOf(&n).Elem()
	for b.Loop() {
		n = 250

		if out := modifiers.Clamp(v, params); out.Error != nil {
			b.Fatal(out.Error)
		}
	}
}
