// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

func TestBuiltins(t *testing.T) {
	t.Parallel()

	type named string

	blank := ""
	vals := []*string{nil, &blank}
	v := struct {
		A named
		B *string
		C string
		D string
		E []string
		F []*string
	}{A: " HEY ", B: &blank, C: "a\u202eb", D: "x\ny", E: []string{"a", " ", "b"}, F: vals}
	r := modifiers.DefaultRegistry()

	cases := []struct {
		name  string
		fn    modifiers.Modifier
		field reflect.Value
	}{{"trim", must(r.Get("trim")), reflect.ValueOf(&v.A).Elem()}, {"nil", must(r.Get("nil_on_empty")), reflect.ValueOf(&v.B).Elem()}, {"bad", must(r.Get("remove_bad_symbols")), reflect.ValueOf(&v.C).Elem()}, {"controls", must(r.Get("remove_bad_symbols")), reflect.ValueOf(&v.D).Elem()}, {"slice", must(r.Get("remove_empty_elements")), reflect.ValueOf(&v.E).Elem()}, {"ptr-slice", must(r.Get("remove_empty_elements")), reflect.ValueOf(&v.F).Elem()}}
	for _, c := range cases {
		if out := c.fn(c.field, nil); out.Err != nil {
			t.Fatal(out.Err)
		}
	}

	if v.A != "HEY" || v.B != nil || v.C != "ab" || v.D != "xy" || len(v.E) != 2 || len(v.F) != 0 {
		t.Fatalf("unexpected result: %#v", v)
	}

	lower, _ := r.Get("lowercase")
	s := "UP"
	lower(reflect.ValueOf(&s).Elem(), nil)

	if s != "up" {
		t.Fatal(s)
	}

	upper, _ := r.Get("uppercase")
	upper(reflect.ValueOf(&s).Elem(), nil)

	if s != "UP" {
		t.Fatal(s)
	}

	if _, ok := modifiers.NewRegistry().Get("absent"); ok {
		t.Fatal("unexpected modifier")
	}

	if strings.TrimSpace(" x ") != "x" {
		t.Fatal("unreachable")
	}
}
func must(m modifiers.Modifier, ok bool) modifiers.Modifier {
	if !ok {
		panic("missing builtin")
	}

	return m
}
func TestParse(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"bad(", "(x=1)"} {
		_ = modifiers.Parse(in)
	}

	if got := modifiers.Parse("phone(region=us; country=ca)"); got.Name != "phone" || !reflect.DeepEqual(got.Params, map[string]string{"region": "us", "country": "ca"}) {
		t.Fatalf("unexpected parsed parameters: %#v", got)
	}

	if got := modifiers.Parse("phone()"); got.Name != "phone" || got.Params == nil || len(got.Params) != 0 {
		t.Fatalf("empty parameters should produce an empty map: %#v", got)
	}

	if got := modifiers.Parse("trim"); got.Name != "trim" || got.Params != nil {
		t.Fatalf("plain modifier should have nil parameters: %#v", got)
	}
}
func TestRegistryRegister(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()
	r.Register("x", func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Value: v}
	})

	if _, ok := r.Get("x"); !ok {
		t.Fatal("not registered")
	}

	if err := r.Register("", nil); err == nil {
		t.Fatal("accepted invalid entry")
	}
}

func TestRegistryZeroValueAndReplacement(t *testing.T) {
	t.Parallel()

	var r modifiers.Registry
	if _, ok := r.Get("x"); ok {
		t.Fatal("empty registry returned a modifier")
	}

	first := modifiers.Modifier(func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Value: reflect.ValueOf("first")}
	})
	second := modifiers.Modifier(func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Value: reflect.ValueOf("second")}
	})

	r.Register("x", first)
	r.Register("y", first)
	r.Register("x", second)

	for name, want := range map[string]string{"x": "second", "y": "first"} {
		m, ok := r.Get(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}

		if got := m(reflect.Value{}, nil).Value.String(); got != want {
			t.Fatalf("Get(%q) resolved to %q, want %q", name, got, want)
		}
	}
}

func TestModifierEdgeCases(t *testing.T) {
	t.Parallel()

	transform := func(s string) (string, bool) { return strings.ToUpper(s), true }

	unchanged := reflect.ValueOf(" ")
	if got := modifiers.ApplyString(unchanged, transform); got.Value.String() != " " {
		t.Fatal(got)
	}

	ptrValue := "lower"
	holder := &ptrValue
	ptr := reflect.ValueOf(&holder).Elem()
	modifiers.ApplyString(ptr, transform)

	if *holder != "LOWER" {
		t.Fatal(ptrValue)
	}

	var nilPtr *string
	modifiers.ApplyString(reflect.ValueOf(nilPtr), transform)
	modifiers.ApplyString(reflect.ValueOf(3), transform)

	s := "value"
	modifiers.NilOnEmpty(reflect.ValueOf(&s).Elem(), nil)

	var p *string
	modifiers.NilOnEmpty(reflect.ValueOf(&p).Elem(), nil)
	modifiers.RemoveEmptyElements(reflect.ValueOf(2), nil)

	array := []int{1}
	modifiers.RemoveEmptyElements(reflect.ValueOf(&array).Elem(), nil)

	nonSettable := []string{"x"}
	modifiers.RemoveEmptyElements(reflect.ValueOf(nonSettable), nil)

	for _, input := range []string{"(bad)", "trim(", "trim()", " trim ", "phone(a=1;flag)"} {
		_ = modifiers.Parse(input)
	}

	var registry *modifiers.Registry
	if _, ok := registry.Get("x"); ok {
		t.Fatal("nil registry returned a modifier")
	}

	if err := registry.Register("x", func(reflect.Value, map[string]string) modifiers.Result { return modifiers.Result{} }); err == nil {
		t.Fatal("a nil registry accepted a registration")
	}
}

func TestRegistryAndParserConcurrent(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				r.Register("trim", func(v reflect.Value, _ map[string]string) modifiers.Result {
					return modifiers.Result{Value: v}
				})
				_, _ = r.Get("trim")
				p := modifiers.Parse("phone(region=US)")
				p.Params["region"] = "XX"
			}
		})
	}

	wg.Wait()

	if got := modifiers.Parse("phone(region=US)").Params["region"]; got != "US" {
		t.Fatalf("parsed parameters are shared between calls: %q", got)
	}
}

func TestApplyStringModifierPreservesNamedStringTypes(t *testing.T) {
	t.Parallel()

	type named string

	for _, pointer := range []bool{false, true} {
		value := named(" value ")

		v := reflect.ValueOf(value)
		if pointer {
			v = reflect.ValueOf(&value)
		}

		result := modifiers.ApplyString(v, func(s string) (string, bool) {
			return strings.TrimSpace(s), true
		})
		if result.Value.Type() != v.Type() {
			t.Fatalf("type changed: got %s want %s", result.Value.Type(), v.Type())
		}

		got := result.Value
		if pointer {
			got = got.Elem()
		}

		if got.String() != "value" {
			t.Fatalf("got %q", got.String())
		}
	}
}

func TestRemoveEmptyElementsPreservesUnchangedAndNamedPointerSlices(t *testing.T) {
	t.Parallel()

	unchanged := []string{"one", "two"}
	v := reflect.ValueOf(&unchanged).Elem()
	backing := v.Pointer()
	modifiers.RemoveEmptyElements(v, nil)

	if v.Pointer() != backing {
		t.Fatal("unchanged slice backing array was replaced")
	}

	type tag string

	a, empty, b := tag("a"), tag(" "), tag("b")
	tags := []*tag{&a, nil, &empty, &b}
	modifiers.RemoveEmptyElements(reflect.ValueOf(&tags).Elem(), nil)

	if !reflect.DeepEqual(tags, []*tag{&a, &b}) {
		t.Fatalf("named string pointers were not preserved: %#v", tags)
	}
}

func TestModifiersIgnoreInvalidAndUnsupportedValues(t *testing.T) {
	t.Parallel()

	upper := func(s string) (string, bool) { return strings.ToUpper(s), s != strings.ToUpper(s) }
	if got := modifiers.ApplyString(reflect.Value{}, upper); got.Value.IsValid() || got.Err != nil {
		t.Fatalf("invalid value: %+v", got)
	}

	if got := modifiers.NilOnEmpty(reflect.Value{}, nil); got.Value.IsValid() || got.Err != nil {
		t.Fatalf("invalid value: %+v", got)
	}

	already := "UPPER"

	v := reflect.ValueOf(&already)
	if got := modifiers.ApplyString(v, upper); got.Value.Pointer() != v.Pointer() {
		t.Fatal("unchanged non-settable pointer was replaced")
	}
}

func TestNilOnEmptyFollowsPointerChains(t *testing.T) {
	t.Parallel()

	empty, value := "", "x"
	emptyPtr, valuePtr := &empty, &value
	chain := struct{ Empty, Value **string }{Empty: &emptyPtr, Value: &valuePtr}
	modifiers.NilOnEmpty(reflect.ValueOf(&chain.Empty).Elem(), nil)
	modifiers.NilOnEmpty(reflect.ValueOf(&chain.Value).Elem(), nil)

	if *chain.Empty != nil {
		t.Fatal("empty string behind a pointer chain was not set to nil")
	}

	if *chain.Value != valuePtr {
		t.Fatal("non-empty string behind a pointer chain was changed")
	}

	var nilChain **string
	if got := modifiers.NilOnEmpty(reflect.ValueOf(&nilChain).Elem(), nil); !got.Value.IsValid() {
		t.Fatal("nil pointer chain lost its value")
	}
}

func TestParseParameters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		clause string
		want   modifiers.Clause
	}{
		{clause: " trim ", want: modifiers.Clause{Name: "trim"}},
		{clause: "", want: modifiers.Clause{}},
		{clause: "(x=1)", want: modifiers.Clause{Name: "(x=1)"}},
		{clause: "phone(", want: modifiers.Clause{Name: "phone("}},
		{clause: "f(a = 1 ;; flag ; k=v=w)", want: modifiers.Clause{Name: "f", Params: map[string]string{"a": "1", "flag": "", "k": "v=w"}}},
	}
	for _, tc := range cases {
		t.Run(tc.clause, func(t *testing.T) {
			t.Parallel()

			if got := modifiers.Parse(tc.clause); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Parse(%q) = %#v, want %#v", tc.clause, got, tc.want)
			}
		})
	}
}
