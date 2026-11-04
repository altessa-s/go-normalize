// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

func TestClamp(t *testing.T) {
	t.Parallel()
	type level int8
	cases := []struct {
		name   string
		params map[string]string
		input  any
		want   any
	}{
		{name: "int above", params: map[string]string{"min": "1", "max": "100"}, input: 250, want: 100},
		{name: "int below", params: map[string]string{"min": "1", "max": "100"}, input: -3, want: 1},
		{name: "int inside", params: map[string]string{"min": "1", "max": "100"}, input: 42, want: 42},
		{name: "min only", params: map[string]string{"min": "0"}, input: int64(math.MinInt64), want: int64(0)},
		{name: "max only", params: map[string]string{"max": "10"}, input: int64(math.MaxInt64), want: int64(10)},
		{name: "int64 precision", params: map[string]string{"max": "9007199254740993"}, input: int64(math.MaxInt64), want: int64(9007199254740993)},
		{name: "named int8", params: map[string]string{"max": "100"}, input: level(120), want: level(100)},
		{name: "uint", params: map[string]string{"min": "5"}, input: uint(1), want: uint(5)},
		{name: "uint8", params: map[string]string{"max": "200"}, input: uint8(255), want: uint8(200)},
		{name: "float", params: map[string]string{"min": "0", "max": "1"}, input: 1.5, want: 1.0},
		{name: "float32", params: map[string]string{"min": "-0.5"}, input: float32(-2), want: float32(-0.5)},
		{name: "infinity", params: map[string]string{"max": "1e9"}, input: math.Inf(1), want: 1e9},
		{name: "string ignored", params: map[string]string{"max": "1"}, input: "99", want: "99"},
	}
	clamp := must(modifiers.DefaultRegistry().Get("clamp"))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := reflect.New(reflect.TypeOf(tc.input)).Elem()
			v.Set(reflect.ValueOf(tc.input))
			if out := clamp(v, tc.params); out.Error != nil {
				t.Fatal(out.Error)
			}
			if got := v.Interface(); got != tc.want {
				t.Fatalf("clamp(%v) = %v (%T), want %v (%T)", tc.input, got, got, tc.want, tc.want)
			}
		})
	}
}

func TestClampPointersAndNaN(t *testing.T) {
	t.Parallel()
	params := map[string]string{"min": "0", "max": "10"}
	n := 20
	v := struct {
		Pointer *int
		Nil     *int
		NaN     float64
	}{Pointer: &n, NaN: math.NaN()}
	for i := range reflect.TypeOf(v).NumField() {
		if out := modifiers.Clamp(reflect.ValueOf(&v).Elem().Field(i), params); out.Error != nil {
			t.Fatal(out.Error)
		}
	}
	if n != 10 || v.Nil != nil || !math.IsNaN(v.NaN) {
		t.Fatalf("unexpected result: n=%d %+v", n, v)
	}
	original := 20
	out := modifiers.Clamp(reflect.ValueOf(&original), params)
	if out.Value.Elem().Int() != 10 || original != 20 {
		t.Fatalf("non-settable pointer: got %v, original %d", out.Value.Elem(), original)
	}
}

func TestClampRejectsInvalidBounds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		params map[string]string
		input  any
	}{
		{name: "no bounds", params: nil, input: 1},
		{name: "not a number", params: map[string]string{"min": "one"}, input: 1},
		{name: "min above max", params: map[string]string{"min": "5", "max": "1"}, input: 1},
		{name: "outside int8", params: map[string]string{"max": "300"}, input: int8(1)},
		{name: "negative uint", params: map[string]string{"min": "-1"}, input: uint(1)},
		{name: "float NaN", params: map[string]string{"max": "NaN"}, input: 1.0},
		{name: "bad float max", params: map[string]string{"min": "0", "max": "x"}, input: 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := reflect.New(reflect.TypeOf(tc.input)).Elem()
			v.Set(reflect.ValueOf(tc.input))
			out := modifiers.Clamp(v, tc.params)
			if !errors.Is(out.Error, modifiers.ErrInvalidParameter) {
				t.Fatalf("got %v, want ErrInvalidParameter", out.Error)
			}
			if v.Interface() != tc.input {
				t.Fatalf("value changed to %v", v.Interface())
			}
		})
	}
}
