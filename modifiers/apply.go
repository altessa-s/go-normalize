// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import "reflect"

// ApplyString transforms string and pointer-to-string values, named
// string types included. fn reports whether it changed the string.
//
// It follows the same rules as Apply, written out for strings because
// every built-in string modifier runs through it.
func ApplyString(v reflect.Value, fn func(string) (string, bool)) Result {
	if !v.IsValid() {
		return Result{}
	}

	if v.Kind() == reflect.Pointer && !v.CanSet() {
		return replaceChain(v, isString, func(src, dst reflect.Value) bool {
			n, changed := fn(src.String())
			if changed {
				dst.SetString(n)
			}

			return changed
		})
	}

	leaf := v
	for leaf.Kind() == reflect.Pointer {
		if leaf.IsNil() {
			return Result{Value: v}
		}

		leaf = leaf.Elem()
	}

	if leaf.Kind() == reflect.String {
		n, c := fn(leaf.String())
		if c {
			if leaf.CanSet() {
				leaf.SetString(n)
			} else {
				x := reflect.New(leaf.Type()).Elem()
				x.SetString(n)

				return Result{Value: x}
			}
		}

		return Result{Value: v}
	}

	return Result{Value: v}
}

// Apply transforms values of type T and pointers to them. A predeclared
// T such as int, float64 or string also matches named types of the same kind,
// so Apply[int] handles type Age int; any other T must match exactly.
// fn reports whether it changed the value.
//
// Settable values are updated in place. A value that cannot be set, or a
// pointer that cannot be set itself, yields a new value in the result and the
// original memory is left to its owner.
func Apply[T any](v reflect.Value, fn func(T) (T, bool)) Result {
	target := reflect.TypeFor[T]()
	accept := func(t reflect.Type) bool { return matches(t, target) }

	return applyToLeaf(v, accept, func(src, dst reflect.Value) bool {
		// accept admitted src's type, so the conversion always yields a T.
		in, _ := reflect.TypeAssert[T](src.Convert(target))

		out, changed := fn(in)
		if changed {
			dst.Set(reflect.ValueOf(&out).Elem().Convert(dst.Type()))
		}

		return changed
	})
}

// applyToLeaf follows v's pointer chain to the value at its end and, when
// accept admits that value's type, lets update read it from src and write a
// changed value to dst. Settable values are updated in place; otherwise the
// result carries a new value and the original memory is left to its owner.
func applyToLeaf(v reflect.Value, accept func(reflect.Type) bool, update func(src, dst reflect.Value) bool) Result {
	if !v.IsValid() {
		return Result{}
	}

	if v.Kind() == reflect.Pointer && !v.CanSet() {
		return replaceChain(v, accept, update)
	}

	leaf := v
	for leaf.Kind() == reflect.Pointer {
		if leaf.IsNil() {
			return Result{Value: v}
		}

		leaf = leaf.Elem()
	}

	if !accept(leaf.Type()) {
		return Result{Value: v}
	}

	if leaf.CanSet() {
		update(leaf, leaf)
		return Result{Value: v}
	}

	x := reflect.New(leaf.Type()).Elem()
	if update(leaf, x) {
		return Result{Value: x}
	}

	return Result{Value: v}
}

// replaceChain applies update to a copy of the pointer chain v, which cannot be
// set, and returns the copy when the value changed. The memory v points to
// belongs to the caller and is never written.
func replaceChain(v reflect.Value, accept func(reflect.Type) bool, update func(src, dst reflect.Value) bool) Result {
	root, leaf, _, ok := copyChain(v, false)
	if !ok || !accept(leaf.Type()) || !update(leaf, leaf) {
		return Result{Value: v}
	}

	return Result{Value: root}
}

// copyChain copies the pointer chain v and the value at its end into new
// memory, returning the copy and its settable final value. A nil pointer in the
// chain ends the copy with ok false, unless fill is set: then it and the levels
// below it get new zero values and filled reports that.
func copyChain(v reflect.Value, fill bool) (root, leaf reflect.Value, filled, ok bool) {
	root = reflect.New(v.Type()).Elem()

	dst, src := root, v
	for src.Kind() == reflect.Pointer {
		if src.IsNil() {
			if !fill {
				return v, v, false, false
			}

			for dst.Kind() == reflect.Pointer {
				dst.Set(reflect.New(dst.Type().Elem()))
				dst = dst.Elem()
			}

			return root, dst, true, true
		}

		dst.Set(reflect.New(src.Type().Elem()))
		dst, src = dst.Elem(), src.Elem()
	}

	dst.Set(src)

	return root, dst, false, true
}

func isString(t reflect.Type) bool { return t.Kind() == reflect.String }

// matches reports whether values of type t can be handled as target.
func matches(t, target reflect.Type) bool {
	if t == target {
		return true
	}

	predeclared := target.PkgPath() == "" && target.Name() != ""

	return predeclared && isBasic(target.Kind()) && t.Kind() == target.Kind()
}

func isBasic(k reflect.Kind) bool {
	switch k { //nolint:exhaustive // composite kinds are not basic
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return true
	}

	return false
}
