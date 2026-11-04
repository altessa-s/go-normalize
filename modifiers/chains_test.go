// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

// The tests below pass pointer chains as values that cannot be set: the memory
// they point to belongs to the caller and must stay untouched, while the result
// carries the change.

func TestApplyStringModifierCopiesNonSettableChains(t *testing.T) {
	t.Parallel()

	s := "  x  "
	inner := &s

	out := modifiers.ApplyString(reflect.ValueOf(&inner), func(v string) (string, bool) {
		n := strings.TrimSpace(v)
		return n, n != v
	})
	if s != "  x  " || inner != &s {
		t.Fatalf("caller memory changed: %q", s)
	}

	if got := out.Value.Elem().Elem().String(); got != "x" || out.Value.Type() != reflect.TypeFor[**string]() {
		t.Fatalf("result = %q of %s", got, out.Value.Type())
	}

	var nilInner *string
	if out := modifiers.ApplyString(reflect.ValueOf(&nilInner), func(v string) (string, bool) { return v, false }); out.Value.Elem().IsValid() && !out.Value.Elem().IsNil() {
		t.Fatalf("nil inner pointer was filled: %v", out.Value)
	}
}

func TestApplyModifierCopiesNonSettableChains(t *testing.T) {
	t.Parallel()

	n := 200
	inner := &n

	out := modifiers.Apply(reflect.ValueOf(&inner), clampAge)
	if n != 200 || inner != &n {
		t.Fatalf("caller memory changed: %d", n)
	}

	if got := out.Value.Elem().Elem().Int(); got != 150 {
		t.Fatalf("result = %d", got)
	}

	small := 20
	smallInner := &small

	original := reflect.ValueOf(&smallInner)
	if out := modifiers.Apply(original, clampAge); out.Value.Pointer() != original.Pointer() {
		t.Fatal("an unchanged chain was replaced")
	}
}

func TestPointerModifiersCopyNonSettableChains(t *testing.T) {
	t.Parallel()

	empty, zero := "", 0

	emptyPtr, zeroPtr := &empty, &zero
	if out := modifiers.NilOnEmpty(reflect.ValueOf(&emptyPtr), nil); emptyPtr != &empty || !out.Value.Elem().IsNil() {
		t.Fatalf("nil_on_empty: caller pointer %v, result %v", emptyPtr, out.Value.Elem())
	}

	if out := modifiers.NilOnZero(reflect.ValueOf(&zeroPtr), nil); zeroPtr != &zero || !out.Value.Elem().IsNil() {
		t.Fatalf("nil_on_zero: caller pointer %v, result %v", zeroPtr, out.Value.Elem())
	}

	var nilInner *string

	out := modifiers.EmptyOnNil(reflect.ValueOf(&nilInner), nil)
	if nilInner != nil {
		t.Fatal("empty_on_nil wrote through a pointer it cannot set")
	}

	if inner := out.Value.Elem(); inner.IsNil() || inner.Elem().String() != "" {
		t.Fatalf("empty_on_nil result: %v", out.Value)
	}

	full := &empty

	original := reflect.ValueOf(&full)
	if out := modifiers.EmptyOnNil(original, nil); out.Value.Pointer() != original.Pointer() {
		t.Fatal("empty_on_nil replaced a chain without nil pointers")
	}

	value := "x"
	valuePtr := &value

	original = reflect.ValueOf(&valuePtr)
	if out := modifiers.NilOnEmpty(original, nil); out.Value.Pointer() != original.Pointer() {
		t.Fatal("nil_on_empty replaced a chain to a non-empty string")
	}
}

func TestApplyStringModifierLeavesSettableNilPointers(t *testing.T) {
	t.Parallel()

	var field *string

	v := reflect.ValueOf(&field).Elem()
	called := false

	out := modifiers.ApplyString(v, func(s string) (string, bool) { called = true; return s, false })
	if called || field != nil || out.Value.Type() != v.Type() {
		t.Fatalf("nil field: called=%v field=%v", called, field)
	}
}
