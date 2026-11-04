// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

func TestNilOnZero(t *testing.T) {
	t.Parallel()

	type point struct{ X, Y int }

	zero, one := 0, 1
	zeroPtr := &zero

	v := struct {
		Int      *int
		NonZero  *int
		Bool     *bool
		Struct   *point
		Chain    **int
		NilInner **int
		Nil      *int
		Plain    int
	}{Int: new(0), NonZero: &one, Bool: new(false), Struct: &point{}, Chain: &zeroPtr, NilInner: new((*int)(nil))}
	for i := range reflect.TypeOf(v).NumField() {
		modifiers.NilOnZero(reflect.ValueOf(&v).Elem().Field(i), nil)
	}

	if v.Int != nil || v.Bool != nil || v.Struct != nil || *v.Chain != nil {
		t.Fatalf("zero values were kept: %+v", v)
	}

	if v.NonZero != &one || v.NilInner == nil || *v.NilInner != nil || v.Nil != nil {
		t.Fatalf("non-zero or nil values changed: %+v", v)
	}

	if got := modifiers.NilOnZero(reflect.Value{}, nil); got.Value.IsValid() {
		t.Fatal("invalid value became valid")
	}
}

func TestEmptyOnNil(t *testing.T) {
	t.Parallel()

	text := "kept"

	v := struct {
		Text  *string
		Kept  *string
		Chain **int
		Plain string
	}{Kept: &text}
	for i := range reflect.TypeOf(v).NumField() {
		modifiers.EmptyOnNil(reflect.ValueOf(&v).Elem().Field(i), nil)
	}

	if v.Text == nil || *v.Text != "" || v.Kept != &text || v.Chain == nil || *v.Chain == nil || **v.Chain != 0 {
		t.Fatalf("unexpected result: %+v", v)
	}

	var nilText *string

	out := modifiers.EmptyOnNil(reflect.ValueOf(nilText), nil)
	if out.Value.IsNil() || out.Value.Elem().String() != "" {
		t.Fatalf("non-settable nil pointer: %v", out.Value)
	}

	if got := modifiers.EmptyOnNil(reflect.ValueOf(3), nil); got.Value.Int() != 3 {
		t.Fatal("non-pointer value changed")
	}
}

func TestDedupe(t *testing.T) {
	t.Parallel()

	type tag string

	nan := math.NaN()
	a, b := "a", "a"

	cases := []struct {
		name  string
		input any
		want  any
	}{
		{name: "strings", input: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "named strings", input: []tag{"x", "x"}, want: []tag{"x"}},
		{name: "ints", input: []int{3, 1, 3, 2, 1}, want: []int{3, 1, 2}},
		{name: "bools", input: []bool{true, false, true}, want: []bool{true, false}},
		{name: "case sensitive", input: []string{"a", "A"}, want: []string{"a", "A"}},
		{name: "single", input: []string{"a"}, want: []string{"a"}},
		{name: "pointers by identity", input: []*string{&a, &b, &a}, want: []*string{&a, &b, &a}},
		{name: "structs unchanged", input: []struct{ N int }{{1}, {1}}, want: []struct{ N int }{{1}, {1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := reflect.New(reflect.TypeOf(tc.input)).Elem()
			v.Set(reflect.ValueOf(tc.input))
			modifiers.Dedupe(v, nil)

			if !reflect.DeepEqual(v.Interface(), tc.want) {
				t.Fatalf("Dedupe(%v) = %v, want %v", tc.input, v.Interface(), tc.want)
			}
		})
	}

	floats := []float64{nan, nan, 0, math.Copysign(0, -1)}
	v := reflect.ValueOf(&floats).Elem()
	modifiers.Dedupe(v, nil)

	if len(floats) != 3 {
		t.Fatalf("NaNs are never equal and 0 equals -0: got %v", floats)
	}

	unique := []string{"a", "b"}
	backing := reflect.ValueOf(unique).Pointer()
	uv := reflect.ValueOf(&unique).Elem()
	modifiers.Dedupe(uv, nil)

	if uv.Pointer() != backing {
		t.Fatal("a slice without repeats got a new backing array")
	}

	modifiers.Dedupe(reflect.ValueOf([]string{"a", "a"}), nil)
	modifiers.Dedupe(reflect.Value{}, nil)
}
