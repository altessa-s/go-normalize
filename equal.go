// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"math"
	"reflect"
)

type valuePair struct {
	a, b uintptr
	typ  reflect.Type
}

// equalValue compares values for normalization changes. Unlike reflect.DeepEqual,
// it treats NaNs and non-nil functions as unchanged when the traversal did not
// replace them, which matters for map values keyed by non-reflexive keys.
func equalValue(a, b reflect.Value, seen map[valuePair]bool) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() { //nolint:exhaustive // remaining kinds compare via Interface where possible
	case reflect.Interface:
		if !sameNil(a, b) {
			return false
		}
		if a.IsNil() {
			return true
		}
		return equalValue(a.Elem(), b.Elem(), seen)
	case reflect.Pointer:
		return equalPointers(a, b, seen)
	case reflect.Func:
		// Function values compare by code pointer. Closures of one literal share it,
		// and since Go 1.27 the compiler may share code between literals, so a
		// modifier that swaps one closure for another can go unnoticed here.
		return a.IsNil() == b.IsNil() && (a.IsNil() || a.Pointer() == b.Pointer())
	case reflect.Float32, reflect.Float64:
		return a.Float() == b.Float() || (math.IsNaN(a.Float()) && math.IsNaN(b.Float()))
	case reflect.Complex64, reflect.Complex128:
		return equalComplex(a.Complex(), b.Complex())
	case reflect.Struct, reflect.Array:
		return equalElements(a, b, seen)
	case reflect.Slice:
		if !sameNil(a, b) {
			return false
		}
		if a.IsNil() {
			return true
		}
		return equalElements(a, b, seen)
	case reflect.Map:
		return equalMaps(a, b, seen)
	}
	if a.CanInterface() && b.CanInterface() {
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
	return a.Equal(b)
}

func sameNil(a, b reflect.Value) bool { return a.IsNil() == b.IsNil() }

func equalPointers(a, b reflect.Value, seen map[valuePair]bool) bool {
	if !sameNil(a, b) {
		return false
	}
	if a.IsNil() || a.Pointer() == b.Pointer() {
		return true
	}
	p := valuePair{a: a.Pointer(), b: b.Pointer(), typ: a.Type()}
	if seen[p] {
		return true
	}
	seen[p] = true
	return equalValue(a.Elem(), b.Elem(), seen)
}

func equalElements(a, b reflect.Value, seen map[valuePair]bool) bool {
	if a.Kind() == reflect.Struct {
		if a.NumField() != b.NumField() {
			return false
		}
		for i := range a.NumField() {
			if !equalValue(a.Field(i), b.Field(i), seen) {
				return false
			}
		}
		return true
	}
	if a.Len() != b.Len() {
		return false
	}
	for i := range a.Len() {
		if !equalValue(a.Index(i), b.Index(i), seen) {
			return false
		}
	}
	return true
}

func equalMaps(a, b reflect.Value, seen map[valuePair]bool) bool {
	if !sameNil(a, b) {
		return false
	}
	if a.IsNil() || a.UnsafePointer() == b.UnsafePointer() {
		return true
	}
	if a.Len() != b.Len() {
		return a.IsNil()
	}
	p := valuePair{a: uintptr(a.UnsafePointer()), b: uintptr(b.UnsafePointer()), typ: a.Type()}
	if seen[p] {
		return true
	}
	seen[p] = true
	iter := a.MapRange()
	for iter.Next() {
		value := b.MapIndex(iter.Key())
		if !value.IsValid() || !equalValue(iter.Value(), value, seen) {
			return false
		}
	}
	return true
}

func equalComplex(a, b complex128) bool {
	return a == b || (sameFloat(real(a), real(b)) && sameFloat(imag(a), imag(b)))
}

func sameFloat(a, b float64) bool { return a == b || (math.IsNaN(a) && math.IsNaN(b)) }
