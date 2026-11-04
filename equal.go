// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"math"
	"reflect"
	"unsafe"
)

type valuePair struct {
	a, b uintptr
	typ  reflect.Type
}

// equalValue compares values for normalization changes. Unlike reflect.DeepEqual,
// it treats NaNs and functions as unchanged when the traversal did not replace
// them, which matters for map values keyed by non-reflexive keys. It may report
// a change that did not happen, never the reverse.
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
		return equalFuncs(a, b)
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

// equalFuncs reports whether two function values are the same closure. Code
// pointers cannot tell: closures of one literal share their code, and since Go
// 1.27 the compiler may share code between literals. A function value points to
// its closure, the code together with the captured variables, so equal closure
// pointers mean the same function. When a closure pointer cannot be read, the
// values count as different, so a change is never lost.
func equalFuncs(a, b reflect.Value) bool {
	if a.IsNil() || b.IsNil() {
		return a.IsNil() == b.IsNil()
	}

	pa, okA := closurePointer(a)
	pb, okB := closurePointer(b)

	return okA && okB && pa == pb
}

// closurePointer returns the closure a function value points to. A value
// without an address, such as the dynamic value of an interface, is copied
// first; the copy points to the same closure. A value read through an
// unexported field without an address cannot be copied and reports false.
func closurePointer(f reflect.Value) (unsafe.Pointer, bool) {
	if !f.CanAddr() {
		if !f.CanInterface() {
			return nil, false
		}

		c := reflect.New(f.Type()).Elem()
		c.Set(f)
		f = c
	}

	return *(*unsafe.Pointer)(f.Addr().UnsafePointer()), true //nolint:gosec // #nosec G103 -- reads the closure pointer a func value holds
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
