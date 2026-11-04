// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"fmt"
	"maps"
	"reflect"

	"github.com/altessa-s/go-normalize/modifiers"
)

// walker holds the state of one Normalize call.
type walker struct {
	source ModifierSource
	// seen holds the pointers, slices and maps on the current traversal path
	// so that cycles end; it is allocated on first use.
	seen map[visit]struct{}
}

type visit struct {
	typ      reflect.Type
	ptr      uintptr
	kind     reflect.Kind
	length   int
	capacity int
}

// enter records key on the traversal path and reports false when it is already there.
func (w *walker) enter(key visit) bool {
	if _, ok := w.seen[key]; ok {
		return false
	}

	if w.seen == nil {
		w.seen = make(map[visit]struct{})
	}

	w.seen[key] = struct{}{}

	return true
}

func (w *walker) leave(key visit) { delete(w.seen, key) }

func (w *walker) walk(v reflect.Value) error {
	switch v.Kind() { //nolint:exhaustive // only structs, collections, pointers, and interfaces need traversal
	case reflect.Interface:
		return w.walkInterface(v)
	case reflect.Pointer:
		if v.IsNil() || !needsWalk(v.Type().Elem()) {
			return nil
		}

		key := visit{typ: v.Type(), ptr: v.Pointer(), kind: reflect.Pointer}
		if !w.enter(key) {
			return nil
		}
		defer w.leave(key)

		return w.walk(v.Elem())
	case reflect.Struct:
		return w.walkStruct(v, true)
	case reflect.Slice:
		return w.walkSlice(v)
	case reflect.Array:
		return w.walkArray(v)
	case reflect.Map:
		return w.walkMap(v)
	}

	return nil
}

func (w *walker) walkInterface(v reflect.Value) error {
	if v.IsNil() || !needsWalk(v.Elem().Type()) {
		return nil
	}

	if !v.CanSet() {
		return w.walk(v.Elem())
	}
	// The dynamic value of an interface is not addressable; normalize a copy and store it back.
	dynamic := reflect.New(v.Elem().Type()).Elem()
	dynamic.Set(v.Elem())

	if err := w.walk(dynamic); err != nil {
		return err
	}

	v.Set(dynamic)

	return nil
}

func (w *walker) walkSlice(v reflect.Value) error {
	if v.Len() == 0 || !needsWalk(v.Type().Elem()) {
		return nil
	}

	key := visit{typ: v.Type(), ptr: v.Pointer(), kind: reflect.Slice, length: v.Len(), capacity: v.Cap()}
	if !w.enter(key) {
		return nil
	}
	defer w.leave(key)

	return w.walkElements(v)
}

func (w *walker) walkArray(v reflect.Value) error {
	if v.Len() == 0 || !needsWalk(v.Type().Elem()) {
		return nil
	}

	return w.walkElements(v)
}

func (w *walker) walkElements(v reflect.Value) error {
	for i := range v.Len() {
		if err := w.walk(v.Index(i)); err != nil {
			return within(err, segment{index: i})
		}
	}

	return nil
}

// walkMap normalizes copies of the map values and writes back only those that
// changed, so normalizing an already normalized map does not write to it.
func (w *walker) walkMap(v reflect.Value) error {
	if v.Len() == 0 || !needsWalk(v.Type().Elem()) {
		return nil
	}

	key := visit{typ: v.Type(), ptr: uintptr(v.UnsafePointer()), kind: reflect.Map}
	if !w.enter(key) {
		return nil
	}
	defer w.leave(key)

	k := reflect.New(v.Type().Key()).Elem()
	elem := reflect.New(v.Type().Elem()).Elem()
	before := reflect.New(v.Type().Elem()).Elem()
	compared := make(map[valuePair]bool)

	iter := v.MapRange()
	for iter.Next() {
		k.SetIterKey(iter)
		elem.SetIterValue(iter)
		before.Set(elem)

		if err := w.walk(elem); err != nil {
			return within(err, segment{name: mapKeySegment})
		}

		clear(compared)

		if equalValue(before, elem, compared) {
			continue
		}
		// A key that is not equal to itself, such as NaN, cannot address its entry again.
		if !k.Equal(k) { //nolint:gocritic // the self-comparison is the NaN check
			return failAt(func(path string) error { return fmt.Errorf("%w at %s", ErrNonReflexiveMapKey, path) })
		}

		v.SetMapIndex(k, elem)
	}

	return nil
}

// walkEmbedded walks an embedded struct, or the struct an embedded pointer
// points to, without calling its CustomNormalizer.
func (w *walker) walkEmbedded(v reflect.Value) error {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		key := visit{typ: v.Type(), ptr: v.Pointer(), kind: reflect.Pointer}
		if !w.enter(key) {
			return nil
		}
		defer w.leave(key)

		v = v.Elem()
	}

	return w.walkStruct(v, false)
}

// walkStruct normalizes the planned fields of v and then, when custom is set,
// calls its CustomNormalizer.
func (w *walker) walkStruct(v reflect.Value, custom bool) error {
	plan := planFor(v.Type())
	for i := range plan.fields {
		f := &plan.fields[i]
		if err := w.walkField(f, v.Field(f.index)); err != nil {
			if f.promoted {
				return err
			}

			return within(err, segment{name: f.name})
		}
	}

	if !custom || !plan.custom || !v.CanAddr() || !v.Addr().CanInterface() {
		return nil
	}

	n, ok := reflect.TypeAssert[CustomNormalizer](v.Addr())
	if !ok {
		return nil
	}

	if err := n.Normalize(); err != nil {
		return failAt(func(path string) error {
			if path == "" {
				return fmt.Errorf("normalize: custom normalizer failed: %w", err)
			}

			return fmt.Errorf("normalize: custom normalizer failed at %s: %w", path, err)
		})
	}

	return nil
}

func (w *walker) walkField(f *fieldPlan, fv reflect.Value) error {
	if len(f.clauses) > 0 {
		if err := w.apply(f, fv); err != nil {
			return err
		}
	}

	switch {
	case f.traverse && f.outerCustom:
		return w.walkEmbedded(fv)
	case f.traverse:
		return w.walk(fv)
	}

	return nil
}

func (w *walker) apply(f *fieldPlan, fv reflect.Value) error {
	for _, clause := range f.clauses {
		m, ok := w.source.Get(clause.Name)
		if !ok {
			name := clause.Name
			return failAt(func(path string) error { return fmt.Errorf("%w: %s at %s", ErrModifierNotFound, name, path) })
		}
		// Modifiers may keep or change their parameters; the plan's copy stays intact.
		out := m(fv, maps.Clone(clause.Params))
		if out.Err != nil {
			return modifierFailure(clause.Name, f.name, out.Err)
		}

		if !out.Value.IsValid() {
			continue
		}

		if !out.Value.Type().AssignableTo(fv.Type()) {
			return modifierFailure(clause.Name, f.name, fmt.Errorf("%w: got %s, want %s", ErrResultType, out.Value.Type(), fv.Type()))
		}

		if fv.CanSet() {
			fv.Set(out.Value)
		}
	}

	return nil
}

// modifierFailure reports err from the named modifier at field as an
// *modifiers.Error whose path is filled in on the way up.
func modifierFailure(modifier, field string, err error) error {
	modifierErr := &modifiers.Error{Modifier: modifier, Field: field, Err: err}

	return failAt(func(path string) error {
		modifierErr.Path = path
		return modifierErr
	})
}
