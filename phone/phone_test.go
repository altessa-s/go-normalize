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

func TestRegisterAndNormalize(t *testing.T) {
	r := modifiers.NewRegistry()
	phone.Register(r)

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

func TestNormalizePhonePointerAndInvalid(t *testing.T) {
	t.Parallel()

	r := modifiers.NewRegistry()
	phone.Register(r)

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
	phone.Register(r)

	type contact struct {
		Phone string `normalize:"phone(region=US)"`
	}

	err := normalize.NormalizeWithRegistry(&contact{Phone: "555-0101"}, r)
	if err == nil || strings.Contains(err.Error(), "555-0101") {
		t.Fatalf("number leaked or error missing: %v", err)
	}
}

func TestNormalizePhoneRegionsAndFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value  string
		region string
		want   string
	}{
		{value: "+7 (916) 123-45-67", want: "+79161234567"},
		{value: "202.555.0123", region: "us", want: "+12025550123"},
	}
	for _, tc := range cases {
		result := phone.NormalizePhone(reflect.ValueOf(tc.value), map[string]string{"region": tc.region})
		if result.Error != nil || result.Value.String() != tc.want {
			t.Fatalf("NormalizePhone(%q, %q) = %q, %v; want %q", tc.value, tc.region, result.Value.String(), result.Error, tc.want)
		}
	}
}

type recordingRegistrar map[string]modifiers.Modifier

func (r recordingRegistrar) Register(name string, modifier modifiers.Modifier) { r[name] = modifier }

func TestRegisterAcceptsAnyRegistrar(t *testing.T) {
	t.Parallel()

	r := recordingRegistrar{}
	phone.Register(r)

	if _, ok := r["phone"]; !ok {
		t.Fatal("phone modifier was not registered")
	}

	phone.Register(nil)
}

func TestNormalizePhoneIgnoresUnsupportedValues(t *testing.T) {
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

			result := phone.NormalizePhone(tc.value, nil)
			if result.Error != nil {
				t.Fatalf("unexpected error: %v", result.Error)
			}

			if result.Value.IsValid() != tc.value.IsValid() {
				t.Fatalf("value validity changed: %v", result.Value)
			}
		})
	}
}

func TestNormalizePhoneNonSettableValues(t *testing.T) {
	t.Parallel()

	params := map[string]string{"region": "US"}

	formatted := phone.NormalizePhone(reflect.ValueOf("+12025550123"), params)
	if formatted.Error != nil || formatted.Value.String() != "+12025550123" {
		t.Fatalf("E.164 input: %v, %v", formatted.Value, formatted.Error)
	}

	raw := "202-555-0123"
	pointer := reflect.ValueOf(&raw)

	result := phone.NormalizePhone(pointer, params)
	if result.Error != nil || result.Value.Elem().String() != "+12025550123" {
		t.Fatalf("pointer input: %v, %v", result.Value, result.Error)
	}

	if raw != "202-555-0123" {
		t.Fatalf("non-settable pointer target was modified: %q", raw)
	}
}

func TestNormalizePhoneCachesFailures(t *testing.T) {
	t.Parallel()

	params := map[string]string{"region": "US"}
	for range 2 {
		result := phone.NormalizePhone(reflect.ValueOf("555-0199"), params)
		if !errors.Is(result.Error, phone.ErrInvalidPhone) {
			t.Fatalf("got %v, want ErrInvalidPhone", result.Error)
		}
	}
}
