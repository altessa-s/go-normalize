// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/altessa-s/go-normalize/modifiers"
)

// TagValueCustom marks fields that are handled only by their parent's CustomNormalizer.
const TagValueCustom = "custom"

// ErrInvalidInput identifies invalid Normalize inputs.
var ErrInvalidInput = errors.New("normalize: input must be a non-nil pointer to a struct")

// ErrModifierNotFound identifies an unregistered tag modifier.
var ErrModifierNotFound = errors.New("normalize: modifier not found")

// ErrNonReflexiveMapKey identifies a map value that changed under a key that
// cannot be looked up again, such as a NaN floating-point key.
var ErrNonReflexiveMapKey = errors.New("normalize: cannot update map value with non-reflexive key")

// CustomNormalizer can provide additional normalization after tagged fields.
type CustomNormalizer interface{ Normalize() error }

// Normalize applies built-in modifiers to fields tagged with normalize.
func Normalize(value any) error { return NormalizeWithRegistry(value, modifiers.DefaultRegistry()) }

// NormalizeWithRegistry applies modifiers from registry; it allows optional adapters such as phone.
func NormalizeWithRegistry(value any, registry *modifiers.Registry) error {
	if value == nil {
		return ErrInvalidInput
	}
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return ErrInvalidInput
	}
	return walk(v.Elem(), "", map[visit]bool{}, registry)
}

type visit struct {
	typ      reflect.Type
	ptr      uintptr
	kind     reflect.Kind
	length   int
	capacity int
}

func walk(v reflect.Value, path string, seen map[visit]bool, r *modifiers.Registry) error {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		if v.CanSet() {
			dynamic := reflect.New(v.Elem().Type()).Elem()
			dynamic.Set(v.Elem())
			if err := walk(dynamic, path, seen, r); err != nil {
				return err
			}
			if dynamic.Type().AssignableTo(v.Type()) {
				v.Set(dynamic)
			}
			return nil
		}
		v = v.Elem()
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Pointer {
			key := visit{typ: v.Type(), ptr: v.Pointer(), kind: reflect.Pointer}
			if seen[key] {
				return nil
			}
			seen[key] = true
			defer delete(seen, key)
		}
		v = v.Elem()
	}
	switch v.Kind() { //nolint:exhaustive // only structs, collections, pointers, and interfaces need traversal
	case reflect.Struct:
		return structFields(v, path, seen, r)
	case reflect.Slice:
		return walkSlice(v, path, seen, r)
	case reflect.Array:
		return walkArray(v, path, seen, r)
	case reflect.Map:
		return walkMap(v, path, seen, r)
	}
	return nil
}

func walkSlice(v reflect.Value, path string, seen map[visit]bool, r *modifiers.Registry) error {
	if v.Len() > 0 {
		key := visit{typ: v.Type(), ptr: v.Pointer(), kind: reflect.Slice, length: v.Len(), capacity: v.Cap()}
		if seen[key] {
			return nil
		}
		seen[key] = true
		defer delete(seen, key)
	}
	return walkArray(v, path, seen, r)
}

func walkArray(v reflect.Value, path string, seen map[visit]bool, r *modifiers.Registry) error {
	for i := 0; i < v.Len(); i++ {
		if err := walk(v.Index(i), fmt.Sprintf("%s[%d]", path, i), seen, r); err != nil {
			return err
		}
	}
	return nil
}

func walkMap(v reflect.Value, path string, seen map[visit]bool, r *modifiers.Registry) error {
	if !v.IsNil() {
		key := visit{typ: v.Type(), ptr: uintptr(v.UnsafePointer()), kind: reflect.Map}
		if seen[key] {
			return nil
		}
		seen[key] = true
		defer delete(seen, key)
	}
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		x := reflect.New(v.Type().Elem()).Elem()
		x.Set(iter.Value())
		before := reflect.New(v.Type().Elem()).Elem()
		before.Set(x)
		if err := walk(x, join(path, "{key}"), seen, r); err != nil {
			return err
		}
		if equalValue(before, x, make(map[valuePair]bool)) {
			continue
		}
		keyCopy := reflect.New(key.Type()).Elem()
		keyCopy.Set(key)
		if !key.Equal(keyCopy) {
			return fmt.Errorf("%w at %s", ErrNonReflexiveMapKey, path)
		}
		v.SetMapIndex(key, x)
	}
	return nil
}

func structFields(v reflect.Value, path string, seen map[visit]bool, r *modifiers.Registry) error {
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		fv := v.Field(i)
		if err := normalizeField(f, fv, path, seen, r); err != nil {
			return err
		}
	}
	if v.CanAddr() && v.Addr().CanInterface() {
		if n, ok := reflect.TypeAssert[CustomNormalizer](v.Addr()); ok {
			if err := n.Normalize(); err != nil {
				if path == "" {
					return fmt.Errorf("normalize: custom normalizer failed: %w", err)
				}
				return fmt.Errorf("normalize: custom normalizer failed at %s: %w", path, err)
			}
		}
	}
	return nil
}

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
		for i := 0; i < a.NumField(); i++ {
			if !equalValue(a.Field(i), b.Field(i), seen) {
				return false
			}
		}
		return true
	}
	if a.Len() != b.Len() {
		return false
	}
	for i := 0; i < a.Len(); i++ {
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

func normalizeField(f reflect.StructField, fv reflect.Value, path string, seen map[visit]bool, r *modifiers.Registry) error {
	tag := strings.TrimSpace(f.Tag.Get("normalize"))
	if !f.IsExported() || !fv.CanInterface() || tag == "-" || tag == TagValueCustom {
		return nil
	}
	fp := join(path, f.Name)
	if tag != "" {
		if err := applyModifiers(f, fv, fp, tag, r); err != nil {
			return err
		}
	}
	return walk(fv, fp, seen, r)
}

func applyModifiers(f reflect.StructField, fv reflect.Value, path, tag string, r *modifiers.Registry) error {
	for clause := range strings.SplitSeq(tag, ",") {
		p := modifiers.Parse(clause)
		if p.Name == "" {
			continue
		}
		m, ok := r.Get(p.Name)
		if !ok {
			return fmt.Errorf("%w: %s at %s", ErrModifierNotFound, p.Name, path)
		}
		out := m(fv, p.Params)
		if out.Error != nil {
			// Modifier errors belong to the modifier. Copy before adding field context
			// so a modifier can safely reuse a sentinel error across calls.
			modifierErr := *out.Error
			if modifierErr.ModifierName == "" {
				modifierErr.ModifierName = p.Name
			}
			modifierErr.FieldName, modifierErr.FieldPath = f.Name, path
			return &modifierErr
		}
		if out.Value.IsValid() && fv.CanSet() && out.Value.Type().AssignableTo(fv.Type()) {
			fv.Set(out.Value)
		}
	}
	return nil
}
func join(p, s string) string {
	if p == "" {
		return s
	}
	return p + "." + s
}
