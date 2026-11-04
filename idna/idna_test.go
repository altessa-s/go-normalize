// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package idna_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/idna"
	"github.com/altessa-s/go-normalize/modifiers"
)

func TestToASCII(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{input: "\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444", want: "xn--e1afmkfd.xn--p1ai"},
		{input: "B\u00fccher.DE", want: "xn--bcher-kva.de"},
		{input: "Example.COM", want: "example.com"},
		{input: "example.com", want: "example.com"},
		{input: "User.Name@B\u00fccher.de", want: "User.Name@xn--bcher-kva.de"},
		{input: "\"a@b\"@m\u00fcnchen.de", want: "\"a@b\"@xn--mnchen-3ya.de"},
		{input: "https://B\u00fccher.de:8443/Pfad?q=1#top", want: "https://xn--bcher-kva.de:8443/Pfad?q=1#top"},
		{input: "https://user@\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444/", want: "https://user@xn--e1afmkfd.xn--p1ai/"},
		{input: "http://[2001:db8::1]:80/", want: "http://[2001:db8::1]:80/"},
		{input: "http://192.0.2.1/", want: "http://192.0.2.1/"},
		{input: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			s := tc.input
			if out := idna.ToASCII(reflect.ValueOf(&s).Elem(), nil); out.Error != nil {
				t.Fatal(out.Error)
			}

			if s != tc.want {
				t.Fatalf("idna(%q) = %q, want %q", tc.input, s, tc.want)
			}
		})
	}
}

func TestToASCIIRejectsInvalidDomains(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"not a domain", "user@exa mple.com", "https://bad_host.example/", "xn--zz.example"} {
		s := input

		out := idna.ToASCII(reflect.ValueOf(&s).Elem(), nil)
		if !errors.Is(out.Error, idna.ErrInvalidDomain) {
			t.Fatalf("idna(%q): got %v, want ErrInvalidDomain", input, out.Error)
		}

		if s != input || strings.Contains(out.Error.Error(), input) {
			t.Fatalf("idna(%q): value changed to %q or leaked into %v", input, s, out.Error)
		}
	}
}

func TestRegisterThroughNormalize(t *testing.T) {
	t.Parallel()
	idna.Register(nil)

	r := modifiers.NewRegistry()

	for _, name := range []string{"trim", "email", "url"} {
		m, _ := modifiers.DefaultRegistry().Get(name)
		r.Register(name, m)
	}

	idna.Register(r)

	type contact struct {
		Email   *string `normalize:"trim,email,idna"`
		Website string  `normalize:"trim,url,idna"`
	}

	c := contact{Email: new(" Anna@\u041f\u0440\u0438\u043c\u0435\u0440.\u0420\u0424 "), Website: " HTTPS://B\u00fccher.DE:443 "}
	if err := normalize.NormalizeWithRegistry(&c, r); err != nil {
		t.Fatal(err)
	}

	if *c.Email != "Anna@xn--e1afmkfd.xn--p1ai" || c.Website != "https://xn--bcher-kva.de/" {
		t.Fatalf("got %q, %q", *c.Email, c.Website)
	}

	bad := contact{Website: "https://bad_host.example"}

	err := normalize.NormalizeWithRegistry(&bad, r)
	if modifierErr, ok := errors.AsType[*modifiers.ModifierError](err); !ok || modifierErr.FieldPath != "Website" || !errors.Is(err, idna.ErrInvalidDomain) {
		t.Fatalf("unexpected error: %v", err)
	}
}
