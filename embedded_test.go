// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
)

type audit struct {
	CreatedBy string `normalize:"trim,lowercase"`
	note      string //nolint:unused // proves unexported fields of an embedded struct stay untouched
}

type address struct {
	City string `normalize:"trim"`
	geo
}

type geo struct {
	Country string `normalize:"uppercase"`
}

type node struct {
	*node

	Name string `normalize:"trim"`
}

func TestNormalizeDescendsIntoUnexportedEmbeddedStructs(t *testing.T) {
	t.Parallel()

	type record struct {
		audit
		*address

		Title string `normalize:"trim"`
	}

	r := record{
		CreatedBy: "  ADMIN ", note: " keep ",
		address: &address{City: " Belgrade ", Country: "rs"},
		Title:   " t ",
	}
	if err := normalize.Normalize(&r); err != nil {
		t.Fatal(err)
	}

	if r.CreatedBy != "admin" || r.note != " keep " || r.City != "Belgrade" || r.Country != "RS" || r.Title != "t" {
		t.Fatalf("got %+v, %+v", r.audit, *r.address)
	}

	empty := record{}
	if err := normalize.Normalize(&empty); err != nil || empty.address != nil {
		t.Fatalf("nil embedded pointer: %v, %+v", err, empty)
	}
}

func TestNormalizeSkipsUnexportedEmbeddedStructsTaggedOut(t *testing.T) {
	t.Parallel()

	type label string

	type record struct {
		audit `normalize:"-"`
		label
	}

	r := record{CreatedBy: " ADMIN ", label: " x "}
	if err := normalize.Normalize(&r); err != nil {
		t.Fatal(err)
	}

	if r.CreatedBy != " ADMIN " || r.label != " x " {
		t.Fatalf("got %+v", r)
	}
}

func TestNormalizeStopsAtCyclesThroughUnexportedEmbeds(t *testing.T) {
	t.Parallel()

	n := &node{Name: " a "}
	n.node = n

	if err := normalize.Normalize(n); err != nil {
		t.Fatal(err)
	}

	if n.Name != "a" || n.node != n {
		t.Fatalf("got %q, self reference kept: %v", n.Name, n.node == n)
	}
}

func TestErrorPathOfPromotedFieldOmitsEmbeddedType(t *testing.T) {
	t.Parallel()

	rejected := errors.New("rejected")
	r := modifiers.NewRegistry()
	_ = r.Register("reject", func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Err: rejected}
	})

	type inner struct {
		Code string `normalize:"reject"`
	}

	type wrapper struct{ *inner }

	type outer struct {
		Items []wrapper
	}

	err := normalize.NormalizeWithRegistry(&outer{Items: []wrapper{{inner: &inner{}}}}, r)

	modifierErr, ok := errors.AsType[*modifiers.Error](err)
	if !ok || modifierErr.Path != "Items[0].Code" || !errors.Is(err, rejected) {
		t.Fatalf("got %v", err)
	}
}

type Counted struct {
	Name  string `normalize:"trim"`
	calls int
}

func (c *Counted) Normalize() error {
	c.calls++
	return nil
}

type overriding struct {
	Counted

	own int
}

func (o *overriding) Normalize() error {
	o.own++
	return nil
}

func TestPromotedCustomNormalizerRunsOnce(t *testing.T) {
	t.Parallel()

	promoted := struct{ Counted }{Name: " a "}
	if err := normalize.Normalize(&promoted); err != nil {
		t.Fatal(err)
	}

	if promoted.calls != 1 || promoted.Name != "a" {
		t.Fatalf("promoted Normalize ran %d times, Name %q", promoted.calls, promoted.Name)
	}

	byPointer := struct{ *Counted }{Counted: &Counted{Name: " b "}}
	if err := normalize.Normalize(&byPointer); err != nil {
		t.Fatal(err)
	}

	if byPointer.calls != 1 || byPointer.Name != "b" {
		t.Fatalf("promoted Normalize through a pointer ran %d times, Name %q", byPointer.calls, byPointer.Name)
	}

	o := overriding{Name: " c "}
	if err := normalize.Normalize(&o); err != nil {
		t.Fatal(err)
	}

	if o.own != 1 || o.calls != 0 || o.Name != "c" {
		t.Fatalf("override ran %d times, embedded %d times, Name %q", o.own, o.calls, o.Name)
	}

	plain := struct{ Item Counted }{Item: Counted{Name: " d "}}
	if err := normalize.Normalize(&plain); err != nil {
		t.Fatal(err)
	}

	if plain.Item.calls != 1 {
		t.Fatalf("a named struct field ran Normalize %d times", plain.Item.calls)
	}
}

type trimmedBase struct {
	Name string `normalize:"trim"`
}

type TrimmedBase struct {
	Name string `normalize:"trim"`
}

type deepEmbed struct{ trimmedBase }

type customValueEmbed struct{ trimmedBase }

func (*customValueEmbed) Normalize() error { return nil }

type customPointerEmbed struct{ *trimmedBase }

func (*customPointerEmbed) Normalize() error { return nil }

// TestPromotedFieldValuesChange asserts that the promoted fields of embedded
// structs are written in place for value and pointer embeds of unexported and
// exported types, however the outer struct is reached.
func TestPromotedFieldValuesChange(t *testing.T) {
	t.Parallel()

	valueEmbed := struct{ trimmedBase }{trimmedBase{" value "}}
	pointerEmbed := struct{ *trimmedBase }{&trimmedBase{" pointer "}}
	exportedValue := struct{ TrimmedBase }{TrimmedBase{" exported value "}}
	exportedPointer := struct{ *TrimmedBase }{&TrimmedBase{" exported pointer "}}
	deep := struct{ deepEmbed }{deepEmbed{trimmedBase{" deep "}}}
	customValue := customValueEmbed{trimmedBase{" custom value "}}
	customPointer := customPointerEmbed{&trimmedBase{" custom pointer "}}
	inSlice := struct{ Items []struct{ trimmedBase } }{Items: []struct{ trimmedBase }{{trimmedBase{" slice "}}}}
	inMap := struct {
		Items map[string]struct{ trimmedBase }
	}{Items: map[string]struct{ trimmedBase }{"k": {trimmedBase{" map "}}}}
	inInterface := struct{ Item any }{Item: &struct{ *trimmedBase }{&trimmedBase{" interface "}}}

	cases := []struct {
		name  string
		value any
		got   func() string
		want  string
	}{
		{"unexported value embed", &valueEmbed, func() string { return valueEmbed.Name }, "value"},
		{"unexported pointer embed", &pointerEmbed, func() string { return pointerEmbed.Name }, "pointer"},
		{"exported value embed", &exportedValue, func() string { return exportedValue.Name }, "exported value"},
		{"exported pointer embed", &exportedPointer, func() string { return exportedPointer.Name }, "exported pointer"},
		{"nested unexported embeds", &deep, func() string { return deep.Name }, "deep"},
		{"value embed in a CustomNormalizer", &customValue, func() string { return customValue.Name }, "custom value"},
		{"pointer embed in a CustomNormalizer", &customPointer, func() string { return customPointer.Name }, "custom pointer"},
		{"slice element", &inSlice, func() string { return inSlice.Items[0].Name }, "slice"},
		{"map value", &inMap, func() string { return inMap.Items["k"].Name }, "map"},
		{"interface", &inInterface, func() string { return inInterface.Item.(*struct{ *trimmedBase }).Name }, "interface"},
	}
	for _, tc := range cases {
		if err := normalize.Normalize(tc.value); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		if got := tc.got(); got != tc.want {
			t.Fatalf("%s: Name = %q, want %q", tc.name, got, tc.want)
		}
	}
}
