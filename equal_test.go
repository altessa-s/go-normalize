// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"math"
	"reflect"
	"testing"
)

func TestEqualValueComparesCompositeValues(t *testing.T) {
	t.Parallel()
	type value struct {
		Number  float64
		Complex complex128
		Pointer *int
		Slice   []string
		Array   [2]int
		Map     map[string]any
		Func    func()
	}
	makeValue := func() value {
		n := 7
		return value{
			Number:  math.NaN(),
			Complex: complex(math.NaN(), math.NaN()),
			Pointer: &n,
			Slice:   []string{"a", "b"},
			Array:   [2]int{3, 4},
			Map:     map[string]any{"items": []int{5, 6}},
			Func:    func() {},
		}
	}
	left, right := makeValue(), makeValue()
	if !equalValue(reflect.ValueOf(left), reflect.ValueOf(right), make(map[valuePair]bool)) {
		t.Fatal("equivalent composite values were treated as changed")
	}

	right.Array[1]++
	if equalValue(reflect.ValueOf(left), reflect.ValueOf(right), make(map[valuePair]bool)) {
		t.Fatal("different array elements were treated as equal")
	}
}

func TestEqualValueHandlesNilAndDistinctMaps(t *testing.T) {
	t.Parallel()
	var nilMap map[string]int
	if !equalValue(reflect.ValueOf(nilMap), reflect.ValueOf(nilMap), make(map[valuePair]bool)) {
		t.Fatal("nil maps should compare equal")
	}
	if equalValue(reflect.ValueOf(nilMap), reflect.ValueOf(map[string]int{}), make(map[valuePair]bool)) {
		t.Fatal("nil and empty maps should compare unequal")
	}
	left := map[string]any{"value": []int{1, 2}}
	right := map[string]any{"value": []int{1, 2}}
	if !equalValue(reflect.ValueOf(left), reflect.ValueOf(right), make(map[valuePair]bool)) {
		t.Fatal("distinct maps with equal nested values should compare equal")
	}
	right["value"] = []int{1, 3}
	if equalValue(reflect.ValueOf(left), reflect.ValueOf(right), make(map[valuePair]bool)) {
		t.Fatal("maps with different nested values should compare unequal")
	}
}

func TestEqualValueComparesInterfacesAndCycles(t *testing.T) {
	t.Parallel()
	type node struct {
		Value int
		Next  *node
	}
	left, right := &node{Value: 1}, &node{Value: 1}
	left.Next, right.Next = left, right
	a, b := reflect.ValueOf(any(left)), reflect.ValueOf(any(right))
	if !equalValue(a, b, make(map[valuePair]bool)) {
		t.Fatal("equivalent cyclic values held by interfaces should compare equal")
	}
	right.Value++
	if equalValue(a, b, make(map[valuePair]bool)) {
		t.Fatal("different values held by interfaces should compare unequal")
	}
}

func TestEqualValueEdgeCases(t *testing.T) {
	t.Parallel()
	type pair struct{ A, B any }
	n := 1
	selfMap := map[string]any{}
	selfMap["self"] = selfMap
	otherSelfMap := map[string]any{}
	otherSelfMap["self"] = otherSelfMap
	cases := []struct {
		name string
		a, b reflect.Value
		want bool
	}{
		{name: "both invalid", a: reflect.Value{}, b: reflect.Value{}, want: true},
		{name: "one invalid", a: reflect.Value{}, b: reflect.ValueOf(1), want: false},
		{name: "different types", a: reflect.ValueOf(1), b: reflect.ValueOf("1"), want: false},
		{name: "nil and set interface", a: reflect.ValueOf(pair{}), b: reflect.ValueOf(pair{A: 1}), want: false},
		{name: "nil interfaces", a: reflect.ValueOf(pair{}), b: reflect.ValueOf(pair{}), want: true},
		{name: "nil and set pointer", a: reflect.ValueOf((*int)(nil)), b: reflect.ValueOf(&n), want: false},
		{name: "same pointer", a: reflect.ValueOf(&n), b: reflect.ValueOf(&n), want: true},
		{name: "nil and empty slice", a: reflect.ValueOf([]int(nil)), b: reflect.ValueOf([]int{}), want: false},
		{name: "nil slices", a: reflect.ValueOf([]int(nil)), b: reflect.ValueOf([]int(nil)), want: true},
		{name: "slice lengths", a: reflect.ValueOf([]int{1}), b: reflect.ValueOf([]int{1, 2}), want: false},
		{name: "map lengths", a: reflect.ValueOf(map[int]int{1: 1}), b: reflect.ValueOf(map[int]int{1: 1, 2: 2}), want: false},
		{name: "missing map key", a: reflect.ValueOf(map[int]int{1: 1}), b: reflect.ValueOf(map[int]int{2: 1}), want: false},
		{name: "cyclic maps", a: reflect.ValueOf(selfMap), b: reflect.ValueOf(otherSelfMap), want: true},
		{name: "nil and set func", a: reflect.ValueOf((func())(nil)), b: reflect.ValueOf(func() {}), want: false},
		{name: "plain values", a: reflect.ValueOf("x"), b: reflect.ValueOf("x"), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := equalValue(tc.a, tc.b, make(map[valuePair]bool)); got != tc.want {
				t.Fatalf("equalValue() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEqualValueComparesUnexportedFields(t *testing.T) {
	t.Parallel()
	type hidden struct{ value int }
	a := reflect.ValueOf(hidden{value: 1})
	if !equalValue(a, reflect.ValueOf(hidden{value: 1}), make(map[valuePair]bool)) {
		t.Fatal("equal unexported fields compared unequal")
	}
	if equalValue(a, reflect.ValueOf(hidden{value: 2}), make(map[valuePair]bool)) {
		t.Fatal("different unexported fields compared equal")
	}
}
