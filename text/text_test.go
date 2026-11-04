// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package text_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/text"
)

func newRegistry() *modifiers.Registry {
	r := modifiers.NewRegistry()
	text.Register(r)
	return r
}

func apply(t *testing.T, r *modifiers.Registry, name string, params map[string]string, input string) string {
	t.Helper()
	m, ok := r.Get(name)
	if !ok {
		t.Fatalf("%s is not registered", name)
	}
	s := input
	if out := m(reflect.ValueOf(&s).Elem(), params); out.Error != nil {
		t.Fatal(out.Error)
	}
	return s
}

func TestModifiers(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	cases := []struct {
		modifier string
		params   map[string]string
		input    string
		want     string
	}{
		{modifier: "nfc", input: "Caf\u00e9", want: "Caf\u00e9"},
		{modifier: "nfc", input: "Cafe\u0301", want: "Caf\u00e9"},
		{modifier: "nfc", input: "\u0438\u0306", want: "\u0439"},
		{modifier: "nfc", input: "\ufb01", want: "\ufb01"},
		{modifier: "nfkc", input: "\ufb01le", want: "file"},
		{modifier: "nfkc", input: "\uff21\uff22\uff23\uff11\uff12\uff13", want: "ABC123"},
		{modifier: "nfkc", input: "x\u00b2", want: "x2"},
		{modifier: "fold", input: "Stra\u00dfe", want: "strasse"},
		{modifier: "fold", input: "STRASSE", want: "strasse"},
		{modifier: "fold", input: "\u03a3\u038a\u03a3\u03a5\u03a6\u039f\u03a3", want: "\u03c3\u03af\u03c3\u03c5\u03c6\u03bf\u03c3"},
		{modifier: "title", input: "hello wORLD", want: "Hello World"},
		{modifier: "title", params: map[string]string{"lang": "nl"}, input: "ijsland", want: "IJsland"},
		{modifier: "title", params: map[string]string{"lang": "tr"}, input: "istanbul", want: "\u0130stanbul"},
		{modifier: "title", input: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.modifier+"/"+tc.input, func(t *testing.T) {
			t.Parallel()
			if got := apply(t, r, tc.modifier, tc.params, tc.input); got != tc.want {
				t.Fatalf("%s(%q) = %q, want %q", tc.modifier, tc.input, got, tc.want)
			}
		})
	}
}

func TestTitleRejectsInvalidLanguage(t *testing.T) {
	t.Parallel()
	s := "hello"
	out := text.Title(reflect.ValueOf(&s).Elem(), map[string]string{"lang": "not a tag"})
	if !errors.Is(out.Error, modifiers.ErrInvalidParameter) || s != "hello" {
		t.Fatalf("got %v, value %q", out.Error, s)
	}
}

func TestRegisterThroughNormalize(t *testing.T) {
	t.Parallel()
	text.Register(nil)
	r := newRegistry()
	type account struct {
		Login *string `normalize:"nfkc,fold"`
		Name  string  `normalize:"nfc,title"`
	}
	a := account{Login: new("\uff21\uff24\uff2d\uff29\uff2e"), Name: "jos\u00e9"}
	if err := normalize.NormalizeWithRegistry(&a, r); err != nil {
		t.Fatal(err)
	}
	if *a.Login != "admin" || a.Name != "Jos\u00e9" {
		t.Fatalf("got %q, %q", *a.Login, a.Name)
	}
}

// TestCasersAreNotShared runs one modifier from many goroutines; a Caser
// shared between calls would race.
func TestCasersAreNotShared(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 200 {
				type v struct {
					S string `normalize:"title(lang=en),fold"`
				}
				x := v{S: "Hello Stra\u00dfe"}
				if err := normalize.NormalizeWithRegistry(&x, r); err != nil || x.S != "hello strasse" {
					t.Errorf("got %q, %v", x.S, err)
					return
				}
			}
		})
	}
	wg.Wait()
}
