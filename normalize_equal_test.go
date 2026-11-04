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
