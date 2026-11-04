// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/altessa-s/go-normalize/modifiers"
)

type age int

func clampAge(n int) (int, bool) {
	if n > 150 {
		return 150, true
	}
	return n, false
}

func TestApplyModifierSettableValues(t *testing.T) {
	t.Parallel()
	v := struct {
		Plain   int
		Named   age
		Pointer **age
		Nil     *int
		Other   string
	}{Plain: 200, Named: 300, Other: "x"}
	inner := age(400)
	innerPtr := &inner
	v.Pointer = &innerPtr
	for _, field := range []string{"Plain", "Named", "Pointer", "Nil", "Other"} {
		f := reflect.ValueOf(&v).Elem().FieldByName(field)
		if out := modifiers.ApplyModifier(f, clampAge); out.Error != nil || out.Value.Type() != f.Type() {
			t.Fatalf("%s: %+v", field, out)
		}
	}
	if v.Plain != 150 || v.Named != 150 || inner != 150 || v.Nil != nil || v.Other != "x" {
		t.Fatalf("unexpected result: %+v, inner=%d", v, inner)
	}
}

func TestApplyModifierNonSettableValues(t *testing.T) {
	t.Parallel()
	out := modifiers.ApplyModifier(reflect.ValueOf(age(200)), clampAge)
	if out.Value.Type() != reflect.TypeFor[age]() || out.Value.Int() != 150 {
		t.Fatalf("value: %v", out.Value)
	}
	unchanged := reflect.ValueOf(age(20))
	if out := modifiers.ApplyModifier(unchanged, clampAge); out.Value.Int() != 20 {
		t.Fatalf("unchanged value: %v", out.Value)
	}
	n := age(200)
	pointer := reflect.ValueOf(&n)
	out = modifiers.ApplyModifier(pointer, clampAge)
	if out.Value.Pointer() == pointer.Pointer() || out.Value.Elem().Int() != 150 || n != 200 {
		t.Fatalf("pointer: got %v, original %d", out.Value.Elem(), n)
	}
	small := age(20)
	if out := modifiers.ApplyModifier(reflect.ValueOf(&small), clampAge); out.Value.Elem().Int() != 20 {
		t.Fatalf("unchanged pointer: %v", out.Value.Elem())
	}
	if out := modifiers.ApplyModifier(reflect.Value{}, clampAge); out.Value.IsValid() {
		t.Fatalf("invalid value: %v", out.Value)
	}
}

func TestApplyModifierMatchesNamedTypesExactly(t *testing.T) {
	t.Parallel()
	type stamp time.Time
	toUTC := func(ts time.Time) (time.Time, bool) { return ts.UTC(), ts.Location() != time.UTC }
	local := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("X", 3600))
	v := struct {
		At    time.Time
		Stamp stamp
	}{At: local, Stamp: stamp(local)}
	modifiers.ApplyModifier(reflect.ValueOf(&v.At).Elem(), toUTC)
	modifiers.ApplyModifier(reflect.ValueOf(&v.Stamp).Elem(), toUTC)
	if v.At.Location() != time.UTC || !v.At.Equal(local) {
		t.Fatalf("time.Time was not converted: %v", v.At)
	}
	if time.Time(v.Stamp).Location() == time.UTC {
		t.Fatal("a named struct type matched time.Time")
	}
}
