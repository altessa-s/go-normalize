// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/phone"
)

// newModifier returns the phone modifier with defaultRegion.
func newModifier(t testing.TB, defaultRegion string) modifiers.Modifier {
	t.Helper()

	m, err := phone.New(defaultRegion)
	if err != nil {
		t.Fatal(err)
	}

	return m
}

func TestRegisterAndNormalize(t *testing.T) {
	r := modifiers.NewRegistry()
	if err := phone.Register(r, "US"); err != nil {
		t.Fatal(err)
	}

	type contact struct {
		Number string `normalize:"phone(region=US)"`
	}

	v := contact{Number: "202-555-0123"}
	if err := normalize.NormalizeWithRegistry(&v, r); err != nil {
		t.Fatal(err)
	}

	if v.Number != "+12025550123" {
		t.Fatalf("got %q", v.Number)
	}
}

func TestPhonePointerAndInvalid(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()
	if err := phone.Register(r, "US"); err != nil {
		t.Fatal(err)
	}

	type contact struct {
		Phone *string `normalize:"phone(region=us)"`
	}

	raw := "202-555-0123"

	v := contact{Phone: &raw}
	if err := normalize.NormalizeWithRegistry(&v, r); err != nil {
		t.Fatal(err)
	}

	if *v.Phone != "+12025550123" {
		t.Fatalf("got %q", *v.Phone)
	}

	bad := contact{Phone: &raw}

	*bad.Phone = "not a phone"
	if err := normalize.NormalizeWithRegistry(&bad, r); err == nil {
		t.Fatal("invalid number accepted")
	}

	empty := contact{}
	if err := normalize.NormalizeWithRegistry(&empty, r); err != nil {
		t.Fatal(err)
	}
}

func TestPhoneErrorDoesNotRetainNumber(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()
	if err := phone.Register(r, "US"); err != nil {
		t.Fatal(err)
	}

	type contact struct {
		Phone string `normalize:"phone(region=US)"`
	}

	err := normalize.NormalizeWithRegistry(&contact{Phone: "555-0101"}, r)
	if err == nil || strings.Contains(err.Error(), "555-0101") {
		t.Fatalf("number leaked or error missing: %v", err)
	}
}

func TestPhoneRegionsAndFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value  string
		region string
		want   string
	}{
		{value: "+7 (916) 123-45-67", want: "+79161234567"},
		{value: "202.555.0123", region: "us", want: "+12025550123"},
	}
	m := newModifier(t, "RU")

	for _, tc := range cases {
		var params map[string]string
		if tc.region != "" {
			params = map[string]string{"region": tc.region}
		}

		result := m(reflect.ValueOf(tc.value), params)
		if result.Err != nil || result.Value.String() != tc.want {
			t.Fatalf("phone(%q, %q) = %q, %v; want %q", tc.value, tc.region, result.Value.String(), result.Err, tc.want)
		}
	}
}

type recordingRegistrar map[string]modifiers.Modifier

func (r recordingRegistrar) Register(name string, modifier modifiers.Modifier) error {
	r[name] = modifier
	return nil
}

func TestRegisterAcceptsAnyRegistrar(t *testing.T) {
	t.Parallel()

	r := recordingRegistrar{}
	if err := phone.Register(r, "us"); err != nil {
		t.Fatal(err)
	}

	if _, ok := r["phone"]; !ok {
		t.Fatal("phone modifier was not registered")
	}

	if err := phone.Register(nil, "US"); !errors.Is(err, modifiers.ErrInvalidRegistration) {
		t.Fatalf("Register(nil) = %v, want ErrInvalidRegistration", err)
	}

	var typedNil *derefRegistrar
	if err := phone.Register(typedNil, "US"); !errors.Is(err, modifiers.ErrInvalidRegistration) {
		t.Fatalf("Register(typed nil) = %v, want ErrInvalidRegistration", err)
	}
}

func TestRegisterRequiresSupportedDefaultRegion(t *testing.T) {
	t.Parallel()

	for _, region := range []string{"", "ZZ", "USA", "001"} {
		r := recordingRegistrar{}
		if err := phone.Register(r, region); !errors.Is(err, phone.ErrUnsupportedRegion) {
			t.Fatalf("Register(%q) = %v, want ErrUnsupportedRegion", region, err)
		}

		if len(r) != 0 {
			t.Fatalf("Register(%q) registered a modifier", region)
		}
	}
}

func TestTagRegionOverridesDefault(t *testing.T) {
	t.Parallel()

	m := newModifier(t, "US")

	result := m(reflect.ValueOf("030 1234567"), map[string]string{"region": " de "})
	if result.Err != nil || result.Value.String() != "+49301234567" {
		t.Fatalf("got %v, %v", result.Value, result.Err)
	}

	for _, region := range []string{"", "ZZ"} {
		result := m(reflect.ValueOf("2025550123"), map[string]string{"region": region})
		if !errors.Is(result.Err, modifiers.ErrInvalidParameter) || !errors.Is(result.Err, phone.ErrUnsupportedRegion) {
			t.Fatalf("region %q: got %v, want ErrInvalidParameter and ErrUnsupportedRegion", region, result.Err)
		}
	}
}

func TestPhoneFollowsPointerChains(t *testing.T) {
	t.Parallel()

	m := newModifier(t, "US")

	raw := "202-555-0123"
	inner := &raw
	outer := &inner

	original := *outer
	if result := m(reflect.ValueOf(&outer).Elem(), nil); result.Err != nil {
		t.Fatal(result.Err)
	}

	if *outer != original || **outer != "+12025550123" {
		t.Fatalf("got %q at %p, want the number rewritten in place at %p", **outer, *outer, original)
	}

	formatted := "+12025550123"
	same := &formatted

	before := same
	if result := m(reflect.ValueOf(&same).Elem(), nil); result.Err != nil || same != before {
		t.Fatalf("an unchanged number replaced the pointer: %v", result.Err)
	}
}

func TestPhoneIgnoresUnsupportedValues(t *testing.T) {
	t.Parallel()

	var nilPhone *string

	cases := []struct {
		name  string
		value reflect.Value
	}{
		{name: "invalid", value: reflect.Value{}},
		{name: "int", value: reflect.ValueOf(42)},
		{name: "nil pointer", value: reflect.ValueOf(nilPhone)},
		{name: "pointer to int", value: reflect.ValueOf(new(42))},
		{name: "empty", value: reflect.ValueOf("")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := newModifier(t, "US")(tc.value, nil)
			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}

			if result.Value.IsValid() != tc.value.IsValid() {
				t.Fatalf("value validity changed: %v", result.Value)
			}
		})
	}
}

func TestPhoneNonSettableValues(t *testing.T) {
	t.Parallel()

	params := map[string]string{"region": "US"}
	m := newModifier(t, "RU")

	formatted := m(reflect.ValueOf("+12025550123"), params)
	if formatted.Err != nil || formatted.Value.String() != "+12025550123" {
		t.Fatalf("E.164 input: %v, %v", formatted.Value, formatted.Err)
	}

	raw := "202-555-0123"
	pointer := reflect.ValueOf(&raw)

	result := m(pointer, params)
	if result.Err != nil || result.Value.Elem().String() != "+12025550123" {
		t.Fatalf("pointer input: %v, %v", result.Value, result.Err)
	}

	if raw != "202-555-0123" {
		t.Fatalf("non-settable pointer target was modified: %q", raw)
	}
}

func TestPhoneCachesFailures(t *testing.T) {
	t.Parallel()

	m := newModifier(t, "US")
	for range 2 {
		result := m(reflect.ValueOf("555-0199"), nil)
		if !errors.Is(result.Err, phone.ErrInvalidPhone) {
			t.Fatalf("got %v, want ErrInvalidPhone", result.Err)
		}
	}
}

// derefRegistrar dereferences its receiver, so calling Register on a typed nil
// one panics.
type derefRegistrar struct{ names []string }

func (r *derefRegistrar) Register(name string, _ modifiers.Modifier) error {
	r.names = append(r.names, name)
	return nil
}
