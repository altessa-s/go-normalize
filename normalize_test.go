// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
)

func TestNormalizeNestedAndCycles(t *testing.T) {
	t.Parallel()

	type child struct {
		Name string `normalize:"trim,lowercase"`
	}

	type node struct {
		Child  *child
		Next   *node
		Values []child
	}

	x := &node{Child: &child{Name: " X "}, Values: []child{{Name: " Y "}}}

	x.Next = x
	if err := normalize.Normalize(x); err != nil {
		t.Fatal(err)
	}

	if x.Child.Name != "x" || x.Values[0].Name != "y" {
		t.Fatalf("not normalized: %#v", x)
	}
}

func TestNormalizeSelfReferentialMapsAndSlices(t *testing.T) {
	t.Parallel()

	type holder struct {
		Map   map[string]any
		Slice []any
	}

	value := holder{Map: make(map[string]any), Slice: make([]any, 1)}
	value.Map["self"] = value.Map

	value.Slice[0] = value.Slice
	if err := normalize.Normalize(&value); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeInterfaceHeldStruct(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `normalize:"trim"`
	}

	type holder struct{ Value any }

	value := holder{Value: item{Name: " x "}}
	if err := normalize.Normalize(&value); err != nil {
		t.Fatal(err)
	}

	got, ok := value.Value.(item)
	if !ok || got.Name != "x" {
		t.Fatalf("interface value was not normalized: %#v", value.Value)
	}
}

func TestNormalizeNonReflexiveMapKey(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `normalize:"trim"`
	}

	type holder struct{ Values map[float64]item }

	nan := math.NaN()
	value := holder{Values: map[float64]item{nan: {Name: " x "}}}

	err := normalize.Normalize(&value)
	if !errors.Is(err, normalize.ErrNonReflexiveMapKey) {
		t.Fatalf("expected non-reflexive key error, got %v", err)
	}

	if len(value.Values) != 1 {
		t.Fatalf("normalization duplicated an unaddressable map entry: %#v", value.Values)
	}

	for _, item := range value.Values {
		if item.Name != " x " {
			t.Fatalf("failed normalization unexpectedly changed original entry: %#v", item)
		}
	}

	unchanged := holder{Values: map[float64]item{nan: {Name: "plain"}}}
	if err := normalize.Normalize(&unchanged); err != nil {
		t.Fatalf("unchanged entry should not require a map write: %v", err)
	}

	if len(unchanged.Values) != 1 {
		t.Fatalf("unchanged NaN-keyed entry was duplicated: %#v", unchanged.Values)
	}

	type odd struct {
		N float64
		F func()
	}

	oddValues := map[float64]odd{nan: {N: math.NaN(), F: func() {}}}
	if err := normalize.Normalize(&struct{ Values map[float64]odd }{Values: oddValues}); err != nil {
		t.Fatalf("unchanged NaN and function values should not require a map write: %v", err)
	}

	if len(oddValues) != 1 {
		t.Fatalf("unchanged NaN-keyed value was duplicated: %#v", oddValues)
	}
}

type failingCustom struct{ cause error }

func (f *failingCustom) Normalize() error { return f.cause }

type customFailure struct{ cause error }

func (e *customFailure) Error() string { return "custom failure" }
func (e *customFailure) Unwrap() error { return e.cause }

func TestCustomNormalizerErrorsIncludeTraversalPath(t *testing.T) {
	t.Parallel()

	want := &customFailure{cause: errors.New("root cause")}
	assertError := func(t *testing.T, err error, path string) {
		t.Helper()

		var got *customFailure
		if !errors.Is(err, want.cause) || !errors.As(err, &got) || got != want || !strings.Contains(err.Error(), path) {
			t.Fatalf("custom error lost path or error identity: %v", err)
		}
	}
	t.Run("nested struct", func(t *testing.T) {
		type parent struct{ Child failingCustom }

		err := normalize.Normalize(&parent{Child: failingCustom{cause: want}})
		assertError(t, err, "Child")
	})
	t.Run("slice element", func(t *testing.T) {
		type parent struct{ Items []failingCustom }

		err := normalize.Normalize(&parent{Items: []failingCustom{{}, {cause: want}}})
		assertError(t, err, "Items[1]")
	})
}

func TestNormalizePointerToPointerStrings(t *testing.T) {
	t.Parallel()

	value := "  hello  "
	inner := &value

	type input struct {
		Text  **string `normalize:"trim,uppercase"`
		Empty **string `normalize:"nil_on_empty"`
	}

	empty := ""
	emptyPointer := &empty

	got := input{Text: &inner, Empty: &emptyPointer}
	if err := normalize.Normalize(&got); err != nil {
		t.Fatal(err)
	}

	if **got.Text != "HELLO" || *got.Empty != nil {
		t.Fatalf("pointer chain was not normalized: %#v", got)
	}
}
func TestNormalizeValidationAndUnknown(t *testing.T) {
	t.Parallel()

	if !errors.Is(normalize.Normalize(nil), normalize.ErrInvalidInput) {
		t.Fatal("nil accepted")
	}

	type input struct {
		X string `normalize:"missing"`
	}

	err := normalize.Normalize(&input{})
	if !errors.Is(err, normalize.ErrModifierNotFound) || !strings.Contains(err.Error(), "X") {
		t.Fatalf("missing modifier field path: %v", err)
	}
}
func TestRegistryCustomModifier(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()
	r.Register("set", func(v reflect.Value, _ map[string]string) modifiers.Result {
		v.SetString("ok")
		return modifiers.Result{Value: v}
	})

	type input struct {
		X string `normalize:"set"`
	}

	v := input{}
	if err := normalize.NormalizeWithRegistry(&v, r); err != nil || v.X != "ok" {
		t.Fatalf("%q %v", v.X, err)
	}
}

func TestNormalizeParameterizedModifiersInSlices(t *testing.T) {
	t.Parallel()

	var captured map[string]string

	r := modifiers.NewRegistry()
	r.Register("capture", func(v reflect.Value, params map[string]string) modifiers.Result {
		captured = params
		return modifiers.Result{Value: v}
	})

	type item struct {
		Value string `normalize:"capture(a=1; flag; c = x=y)"`
		Note  string `normalize:"custom"`
	}

	type holder struct{ Items []item }

	values := holder{Items: []item{{Value: "a", Note: "lower"}, {Value: "b", Note: "upper"}}}
	if err := normalize.NormalizeWithRegistry(&values, r); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(captured, map[string]string{"a": "1", "flag": "", "c": "x=y"}) {
		t.Fatalf("unexpected parsed parameters: %#v", captured)
	}
}

type customRecord struct {
	Name  string `normalize:"trim"`
	Extra string `normalize:"custom"`
	Calls int
}

func (r *customRecord) Normalize() error {
	r.Calls++
	r.Extra = strings.TrimSpace(r.Extra)

	return nil
}

func TestCustomNormalizerRunsAfterTaggedFields(t *testing.T) {
	t.Parallel()

	values := struct{ Items []customRecord }{
		Items: []customRecord{{Name: " name ", Extra: " extra "}, {Name: " other ", Extra: " value "}},
	}
	if err := normalize.Normalize(&values); err != nil {
		t.Fatal(err)
	}

	want := []customRecord{{Name: "name", Extra: "extra", Calls: 1}, {Name: "other", Extra: "value", Calls: 1}}
	for i, got := range values.Items {
		if got != want[i] {
			t.Fatalf("custom normalization did not run for item %d: %#v", i, got)
		}
	}
}
func FuzzParseModifier(f *testing.F) {
	f.Add("phone(region=US)")
	f.Add("trim")
	f.Fuzz(func(t *testing.T, s string) {
		p := modifiers.Parse(s)
		if p.Name == "" {
			return
		}
	})
}

func TestNormalizeReflectionEdges(t *testing.T) {
	t.Parallel()

	type Embedded struct {
		Value string `normalize:"trim,uppercase"`
	}

	type row struct {
		*Embedded
		hidden string `normalize:"missing"` //nolint:unused // proves unexported fields are never resolved
		Text   string `normalize:"trim"`
	}

	type root struct {
		Rows   []*row
		Table  map[int]row
		Array  [1]row
		Nil    **row
		Skip   string `normalize:"-"`
		Custom string `normalize:"custom"`
	}

	r := root{Rows: []*row{{Embedded: &Embedded{Value: " x "}, Text: " y "}, nil}, Table: map[int]row{2: {Text: " z "}}, Array: [1]row{{Text: " q "}}, Skip: " untouched ", Custom: "custom value"}
	if err := normalize.Normalize(&r); err != nil {
		t.Fatal(err)
	}

	if r.Rows[0].Value != "X" || r.Rows[0].Text != "y" || r.Table[2].Text != "z" || r.Array[0].Text != "q" || r.Skip != " untouched " || r.Custom != "custom value" {
		t.Fatalf("unexpected normalization: %#v", r)
	}

	var invalid *struct{ X string }
	if err := normalize.Normalize(invalid); !errors.Is(err, normalize.ErrInvalidInput) {
		t.Fatalf("got %v", err)
	}

	if err := normalize.Normalize(struct{}{}); !errors.Is(err, normalize.ErrInvalidInput) {
		t.Fatalf("got %v", err)
	}
}
func TestNormalizeModifierErrorPathAndCustom(t *testing.T) {
	t.Parallel()

	type item struct {
		Value string `normalize:"reject"`
	}

	type parent struct{ Items []item }

	r := modifiers.NewRegistry()
	r.Register("reject", func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Err: errors.New("bad")}
	})

	err := normalize.NormalizeWithRegistry(&parent{Items: []item{{}}}, r)
	if err == nil || !strings.Contains(err.Error(), "Items[0].Value") {
		t.Fatalf("missing path: %v", err)
	}
}

func TestModifierErrorWrapsSharedSentinelPerField(t *testing.T) {
	t.Parallel()

	rejected := errors.New("rejected")
	r := modifiers.NewRegistry()
	r.Register("reject", func(reflect.Value, map[string]string) modifiers.Result {
		return modifiers.Result{Err: rejected}
	})

	type input struct {
		First string `normalize:"reject"`
	}

	type secondInput struct {
		Second string `normalize:"reject"`
	}

	err := normalize.NormalizeWithRegistry(&input{}, r)

	first, ok := errors.AsType[*modifiers.Error](err)
	if !ok || first.Field != "First" || first.Modifier != "reject" || !errors.Is(err, rejected) {
		t.Fatalf("first field context missing: %v", err)
	}

	err = normalize.NormalizeWithRegistry(&secondInput{}, r)

	second, ok := errors.AsType[*modifiers.Error](err)
	if !ok || second.Field != "Second" || !errors.Is(err, rejected) {
		t.Fatalf("second field context missing: %v", err)
	}

	if first.Field != "First" || first.Path != "First" {
		t.Fatalf("a later call changed an earlier error: %#v", first)
	}
}

func TestPhoneTagRequiresRegistration(t *testing.T) {
	t.Parallel()

	type contact struct {
		Phone string `normalize:"phone(region=US)"`
	}

	err := normalize.Normalize(&contact{Phone: "2025550123"})
	if !errors.Is(err, normalize.ErrModifierNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestMapKeyIsRedactedFromErrorPath(t *testing.T) {
	t.Parallel()

	type value struct {
		Text string `normalize:"missing"`
	}

	input := map[string]value{"private-key": {}}

	err := normalize.Normalize(&struct{ Items map[string]value }{Items: input})
	if err == nil || strings.Contains(err.Error(), "private-key") {
		t.Fatalf("map key leaked or error missing: %v", err)
	}
}

type staticSource map[string]modifiers.Modifier

func (s staticSource) Get(name string) (modifiers.Modifier, bool) {
	m, ok := s[name]
	return m, ok
}

func TestNormalizeWithCustomModifierSource(t *testing.T) {
	t.Parallel()

	source := staticSource{"shout": func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.ApplyString(v, func(s string) (string, bool) { return strings.ToUpper(s) + "!", true })
	}}

	v := struct {
		Text string `normalize:"shout"`
	}{Text: "hi"}
	if err := normalize.NormalizeWithRegistry(&v, source); err != nil {
		t.Fatal(err)
	}

	if v.Text != "HI!" {
		t.Fatalf("got %q", v.Text)
	}
}

func TestNormalizeWithNilRegistry(t *testing.T) {
	t.Parallel()

	untagged := struct{ Text string }{Text: " x "}
	if err := normalize.NormalizeWithRegistry(&untagged, nil); err != nil {
		t.Fatalf("untagged value with nil registry: %v", err)
	}

	tagged := struct {
		Text string `normalize:"trim"`
	}{Text: " x "}
	if err := normalize.NormalizeWithRegistry(&tagged, nil); !errors.Is(err, normalize.ErrModifierNotFound) {
		t.Fatalf("got %v, want ErrModifierNotFound", err)
	}

	var registry *modifiers.Registry
	if err := normalize.NormalizeWithRegistry(&tagged, registry); !errors.Is(err, normalize.ErrModifierNotFound) {
		t.Fatalf("typed nil registry: got %v, want ErrModifierNotFound", err)
	}
}

func TestNormalizeRecursiveElementTypesTerminate(t *testing.T) {
	t.Parallel()

	type pointerChain *pointerChain

	type input struct {
		Chain pointerChain
		Text  string `normalize:"trim"`
	}

	var chain pointerChain

	chain = &chain

	v := input{Chain: chain, Text: " x "}
	if err := normalize.Normalize(&v); err != nil {
		t.Fatal(err)
	}

	if v.Text != "x" {
		t.Fatalf("got %q", v.Text)
	}
}

// TestNormalizeDoesNotWriteUnchangedMaps runs readers next to Normalize on
// already normalized data; the race detector fails it if a map is written.
func TestNormalizeDoesNotWriteUnchangedMaps(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `normalize:"trim"`
	}

	type holder struct{ Items map[string]item }

	items := map[string]item{"a": {Name: "a"}, "b": {Name: "b"}}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				for _, it := range items {
					_ = it.Name
				}
			}
		})
	}

	for range 100 {
		if err := normalize.Normalize(&holder{Items: items}); err != nil {
			t.Error(err)
			break
		}
	}

	wg.Wait()
}

type failingRoot struct {
	Name string `normalize:"trim"`
}

var errRootFailure = errors.New("root failure")

func (*failingRoot) Normalize() error { return errRootFailure }

func TestCustomNormalizerErrorAtRootHasNoPath(t *testing.T) {
	t.Parallel()

	err := normalize.Normalize(&failingRoot{Name: " x "})
	if !errors.Is(err, errRootFailure) {
		t.Fatalf("got %v, want the custom error", err)
	}

	if strings.Contains(err.Error(), " at ") {
		t.Fatalf("root error names a path: %v", err)
	}
}

func TestNormalizeInterfaceValues(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `normalize:"trim"`
	}

	type broken struct {
		Name string `normalize:"missing"`
	}

	type holder struct {
		Nil    any
		Scalar any
		Item   any
		Items  []any
	}

	v := holder{Scalar: " kept ", Item: &item{Name: " a "}, Items: []any{item{Name: " b "}, 7}}
	if err := normalize.Normalize(&v); err != nil {
		t.Fatal(err)
	}

	if v.Nil != nil || v.Scalar != " kept " || v.Item.(*item).Name != "a" || v.Items[0].(item).Name != "b" || v.Items[1] != 7 {
		t.Fatalf("unexpected result: %#v", v)
	}

	err := normalize.Normalize(&holder{Items: []any{broken{}}})
	if !errors.Is(err, normalize.ErrModifierNotFound) || !strings.Contains(err.Error(), "Items[0].Name") {
		t.Fatalf("error inside an interface lost its path: %v", err)
	}
}

func TestNormalizeEmptyContainers(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `normalize:"missing"`
	}

	v := struct {
		Slice    []item
		Array    [0]item
		Map      map[string]item
		NilSlice []item
		NilMap   map[string]*item
		Pointer  *item
	}{Slice: []item{}, Map: map[string]item{}}
	if err := normalize.Normalize(&v); err != nil {
		t.Fatalf("empty containers have nothing to normalize: %v", err)
	}
}

func TestModifierErrorPathThroughMapsAndArrays(t *testing.T) {
	t.Parallel()

	type leaf struct {
		Value string `normalize:"reject"`
	}

	type branch struct{ Leaves [2]leaf }

	r := modifiers.NewRegistry()
	r.Register("reject", func(v reflect.Value, _ map[string]string) modifiers.Result {
		if v.String() == "bad" {
			return modifiers.Result{Err: errors.New("bad value")}
		}

		return modifiers.Result{Value: v}
	})
	v := struct{ Branches map[string]branch }{Branches: map[string]branch{"secret": {Leaves: [2]leaf{{Value: "ok"}, {Value: "bad"}}}}}
	err := normalize.NormalizeWithRegistry(&v, r)

	modifierErr, ok := errors.AsType[*modifiers.Error](err)
	if !ok {
		t.Fatalf("got %v, want a *modifiers.Error", err)
	}

	if modifierErr.Path != "Branches.{key}.Leaves[1].Value" || modifierErr.Field != "Value" || modifierErr.Modifier != "reject" {
		t.Fatalf("unexpected error context: %+v", modifierErr)
	}
}

type closureHolder struct {
	Name  string `normalize:"trim"`
	Value func() int
}

func constant(n int) func() int { return func() int { return n } }

// Normalize replaces Value with another closure of the same function literal;
// both share their code, so only the closure tells them apart.
func (h *closureHolder) Normalize() error {
	h.Value = constant(h.Value() + 1)
	return nil
}

func TestNormalizeWritesBackReplacedClosuresInMaps(t *testing.T) {
	t.Parallel()

	v := struct{ Items map[string]closureHolder }{Items: map[string]closureHolder{"a": {Name: "x", Value: constant(1)}}}
	if err := normalize.Normalize(&v); err != nil {
		t.Fatal(err)
	}

	if got := v.Items["a"].Value(); got != 2 {
		t.Fatalf("replaced closure was not written back: Value() = %d, want 2", got)
	}
}

// TestNormalizeKeepsUnchangedFunctionsInInterfaceMaps covers functions held by
// interface-typed map values, which have no address of their own.
func TestNormalizeKeepsUnchangedFunctionsInInterfaceMaps(t *testing.T) {
	t.Parallel()

	items := map[string]any{"f": func() {}, "g": constant(1)}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				for _, it := range items {
					_ = it
				}
			}
		})
	}

	for range 100 {
		if err := normalize.Normalize(&struct{ Items map[string]any }{Items: items}); err != nil {
			t.Error(err)
			break
		}
	}

	wg.Wait()

	nan := map[float64]any{math.NaN(): func() {}}
	if err := normalize.Normalize(&struct{ Items map[float64]any }{Items: nan}); err != nil {
		t.Fatalf("an unchanged function under a NaN key needs no write: %v", err)
	}
}

func TestModifierResultOfWrongTypeFails(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()
	_ = r.Register("number", func(reflect.Value, map[string]string) modifiers.Result {
		return modifiers.Result{Value: reflect.ValueOf(42)}
	})
	_ = r.Register("keep", func(reflect.Value, map[string]string) modifiers.Result {
		return modifiers.Result{}
	})

	type input struct {
		Kept string `normalize:"keep"`
		Name string `normalize:"number"`
	}

	value := input{Kept: "k", Name: "n"}
	err := normalize.NormalizeWithRegistry(&value, r)

	modifierErr, ok := errors.AsType[*modifiers.Error](err)
	if !ok || !errors.Is(err, normalize.ErrResultType) || modifierErr.Path != "Name" || modifierErr.Modifier != "number" {
		t.Fatalf("got %v, want ErrResultType at Name", err)
	}

	if value.Kept != "k" || value.Name != "n" {
		t.Fatalf("fields changed: %+v", value)
	}
}
