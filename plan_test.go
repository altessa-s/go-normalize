// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"errors"
	"reflect"
	"testing"
)

type recursivePointer *recursivePointer

type recursiveSlice []recursiveSlice

type planCustom struct {
	Name string `normalize:"trim"`
}

func (*planCustom) Normalize() error { return nil }

func TestNeedsWalk(t *testing.T) {
	t.Parallel()

	cases := []struct {
		typ  reflect.Type
		want bool
	}{
		{typ: reflect.TypeFor[string](), want: false},
		{typ: reflect.TypeFor[*string](), want: false},
		{typ: reflect.TypeFor[[]int](), want: false},
		{typ: reflect.TypeFor[[4]float64](), want: false},
		{typ: reflect.TypeFor[map[string][]byte](), want: false},
		{typ: reflect.TypeFor[func()](), want: false},
		{typ: reflect.TypeFor[chan struct{}](), want: false},
		{typ: reflect.TypeFor[struct{}](), want: true},
		{typ: reflect.TypeFor[any](), want: true},
		{typ: reflect.TypeFor[map[string][]*planCustom](), want: true},
		{typ: reflect.TypeFor[recursivePointer](), want: true},
		{typ: reflect.TypeFor[recursiveSlice](), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.typ.String(), func(t *testing.T) {
			t.Parallel()

			if got := needsWalk(tc.typ); got != tc.want {
				t.Fatalf("needsWalk(%s) = %v, want %v", tc.typ, got, tc.want)
			}
		})
	}
}

func TestBuildPlanKeepsOnlyFieldsWithWork(t *testing.T) {
	t.Parallel()

	type input struct {
		Plain    string
		Trimmed  string `normalize:" trim , lowercase ,"`
		Skipped  string `normalize:"-"`
		Custom   string `normalize:"custom"`
		Nested   *planCustom
		Counts   map[string]int
		Empty    string `normalize:","`
		internal string `normalize:"trim"` //nolint:unused // proves unexported fields stay out of the plan
	}

	plan := buildPlan(reflect.TypeFor[input]())
	if plan.custom {
		t.Fatal("input does not implement CustomNormalizer")
	}

	var names []string
	for _, f := range plan.fields {
		names = append(names, f.name)
	}

	if !reflect.DeepEqual(names, []string{"Trimmed", "Nested"}) {
		t.Fatalf("planned fields = %v, want [Trimmed Nested]", names)
	}

	trimmed := plan.fields[0]
	if len(trimmed.clauses) != 2 || trimmed.clauses[0].Name != "trim" || trimmed.clauses[1].Name != "lowercase" || trimmed.traverse {
		t.Fatalf("unexpected plan for Trimmed: %+v", trimmed)
	}

	if !plan.fields[1].traverse || len(plan.fields[1].clauses) != 0 {
		t.Fatalf("unexpected plan for Nested: %+v", plan.fields[1])
	}

	if !buildPlan(reflect.TypeFor[planCustom]()).custom {
		t.Fatal("*planCustom implements CustomNormalizer")
	}

	if planFor(reflect.TypeFor[input]()) != planFor(reflect.TypeFor[input]()) {
		t.Fatal("plans are not cached per type")
	}
}

func TestFormatPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		segments []segment // innermost first
		want     string
	}{
		{name: "empty", want: ""},
		{name: "field", segments: []segment{{name: "Name"}}, want: "Name"},
		{name: "nested", segments: []segment{{name: "Name"}, {index: 12}, {name: "Items"}}, want: "Items[12].Name"},
		{name: "map", segments: []segment{{name: "City"}, {name: mapKeySegment}, {name: "Addresses"}}, want: "Addresses.{key}.City"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := formatPath(tc.segments); got != tc.want {
				t.Fatalf("formatPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPathErrorResolvesWithItsPath(t *testing.T) {
	t.Parallel()

	err := within(failAt(func(path string) error { return errors.New("failed at " + path) }), segment{name: "Name"})
	err = within(err, segment{index: 2})

	err = within(err, segment{name: "Items"})
	if got := err.Error(); got != "failed at Items[2].Name" {
		t.Fatalf("Error() = %q", got)
	}

	if got := resolvePath(err).Error(); got != "failed at Items[2].Name" {
		t.Fatalf("resolvePath() = %q", got)
	}

	plain := errors.New("plain")
	if within(plain, segment{name: "X"}) != plain || resolvePath(plain) != plain {
		t.Fatal("errors without a path must pass through unchanged")
	}
}
