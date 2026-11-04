// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize_test

import (
	"maps"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
)

type benchLeaf struct {
	Value string `normalize:"trim,uppercase"`
}

type benchRecord struct {
	ID      int
	Name    string   `normalize:"trim,lowercase"`
	Email   *string  `normalize:"trim,lowercase,nil_on_empty"`
	Tags    []string `normalize:"remove_empty_elements"`
	Note    string   `normalize:"remove_bad_symbols"`
	Leaf    *benchLeaf
	Skipped string `normalize:"-"`
}

type benchUntagged struct {
	A, B, C, D, E, F, G, H string
	I, J, K, L, M, N, O, P int
}

type benchCustom struct {
	Name  string `normalize:"trim"`
	Extra string `normalize:"custom"`
}

func (c *benchCustom) Normalize() error {
	c.Extra = strings.TrimSpace(c.Extra)
	return nil
}

func benchRecords(n int) []benchRecord {
	records := make([]benchRecord, n)
	for i := range records {
		records[i] = benchRecord{
			ID:    i,
			Name:  "  Name " + strconv.Itoa(i) + "  ",
			Email: new(" USER@EXAMPLE.COM "),
			Tags:  []string{"a", " ", "b"},
			Note:  "note\u202e",
			Leaf:  &benchLeaf{Value: " leaf "},
		}
	}
	return records
}

func BenchmarkNormalizeFlat(b *testing.B) {
	type row struct {
		A string `normalize:"trim,lowercase"`
		B string `normalize:"uppercase"`
	}
	b.ReportAllocs()
	for b.Loop() {
		v := row{A: " x ", B: "y"}
		if err := normalize.Normalize(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeRecord(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		v := benchRecords(1)[0]
		if err := normalize.Normalize(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeUntagged(b *testing.B) {
	b.ReportAllocs()
	v := benchUntagged{A: " a ", P: 1}
	for b.Loop() {
		if err := normalize.Normalize(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeNested(b *testing.B) {
	type item struct {
		Leaf   *benchLeaf
		Values []benchLeaf
	}
	type record struct{ Items []item }
	b.ReportAllocs()
	for b.Loop() {
		v := record{Items: []item{{Leaf: &benchLeaf{Value: " x "}, Values: []benchLeaf{{Value: " y "}}}}}
		if err := normalize.Normalize(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeSlice(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			template := benchRecords(n)
			v := struct{ Items []benchRecord }{Items: make([]benchRecord, n)}
			b.ReportAllocs()
			for b.Loop() {
				copy(v.Items, template)
				if err := normalize.Normalize(&v); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkNormalizeMap(b *testing.B) {
	for _, n := range []int{10, 100} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			template := make(map[string]benchLeaf, n)
			for i := range n {
				template[strconv.Itoa(i)] = benchLeaf{Value: " value "}
			}
			v := struct{ Items map[string]benchLeaf }{Items: make(map[string]benchLeaf, n)}
			b.ReportAllocs()
			for b.Loop() {
				maps.Copy(v.Items, template)
				if err := normalize.Normalize(&v); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkNormalizeScalarMap(b *testing.B) {
	v := struct{ Counts map[string]int }{Counts: make(map[string]int, 100)}
	for i := range 100 {
		v.Counts[strconv.Itoa(i)] = i
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := normalize.Normalize(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeCustomNormalizer(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		v := benchCustom{Name: " name ", Extra: " extra "}
		if err := normalize.Normalize(&v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeWithRegistryParams(b *testing.B) {
	r := modifiers.NewRegistry()
	r.Register("pad", func(v reflect.Value, params map[string]string) modifiers.ModifierResult {
		return modifiers.ApplyStringModifier(v, func(s string) (string, bool) { return s + params["with"], params["with"] != "" })
	})
	type row struct {
		Value string `normalize:"pad(with=!)"`
	}
	b.ReportAllocs()
	for b.Loop() {
		v := row{Value: "x"}
		if err := normalize.NormalizeWithRegistry(&v, r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v := benchRecords(1)[0]
			if err := normalize.Normalize(&v); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
