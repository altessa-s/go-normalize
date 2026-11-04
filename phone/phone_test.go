// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone_test

import (
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
func BenchmarkPhone(b *testing.B) {
	r := modifiers.NewRegistry()
	phone.Register(r)
	type contact struct {
		Phone string `normalize:"phone(region=US)"`
	}
	for b.Loop() {
		v := contact{Phone: "2025550123"}
		if err := normalize.NormalizeWithRegistry(&v, r); err != nil {
			b.Fatal(err)
		}
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
