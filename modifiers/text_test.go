// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

// applyNamed runs the default-registry modifier name on a settable copy of input.
func applyNamed(t *testing.T, name string, params map[string]string, input string) (string, error) {
	t.Helper()

	m, ok := modifiers.DefaultRegistry().Get(name)
	if !ok {
		t.Fatalf("%s is not registered", name)
	}

	s := input
	out := m(reflect.ValueOf(&s).Elem(), params)

	return s, out.Err
}

func TestStringModifiers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		modifier string
		params   map[string]string
		input    string
		want     string
	}{
		{modifier: "collapse_spaces", input: "a  b", want: "a b"},
		{modifier: "collapse_spaces", input: "a\t\n\r b", want: "a b"},
		{modifier: "collapse_spaces", input: "a\u00a0b\u3000c\u0085d", want: "a b c d"},
		{modifier: "collapse_spaces", input: "  a  ", want: " a "},
		{modifier: "collapse_spaces", input: "a b c", want: "a b c"},
		{modifier: "collapse_spaces", input: "", want: ""},
		{modifier: "collapse_spaces", input: "a\u200bb", want: "a\u200bb"},
		{modifier: "collapse_spaces", input: "a\xffb  c", want: "a\xffb c"},
		{modifier: "truncate", params: map[string]string{"max": "3"}, input: "abcdef", want: "abc"},
		{modifier: "truncate", params: map[string]string{"max": "3"}, input: "\u0430\u0431\u0432\u0433", want: "\u0430\u0431\u0432"},
		{modifier: "truncate", params: map[string]string{"max": "3"}, input: "a\U0001f600bc", want: "a\U0001f600b"},
		{modifier: "truncate", params: map[string]string{"max": "3"}, input: "abc", want: "abc"},
		{modifier: "truncate", params: map[string]string{"max": "3"}, input: "\u0430\u0431\u0432", want: "\u0430\u0431\u0432"},
		{modifier: "truncate", params: map[string]string{"max": "0"}, input: "abc", want: ""},
		{modifier: "truncate", params: map[string]string{"max": "2"}, input: "\xff\xfeab", want: "\xff\xfe"},
		{modifier: "digits_only", input: "+7 (916) 123-45-67", want: "79161234567"},
		{modifier: "digits_only", input: "\u0661\u0662\u0663", want: ""},
		{modifier: "digits_only", input: "0123", want: "0123"},
		{modifier: "email", input: "  John.Doe@Example.COM ", want: "John.Doe@example.com"},
		{modifier: "email", input: "\"a@b\"@Example.org", want: "\"a@b\"@example.org"},
		{modifier: "email", input: " no-at-sign ", want: "no-at-sign"},
		{modifier: "email", input: "j@example.com", want: "j@example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.modifier+"/"+tc.input, func(t *testing.T) {
			t.Parallel()

			got, err := applyNamed(t, tc.modifier, tc.params, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if got != tc.want {
				t.Fatalf("%s(%q) = %q, want %q", tc.modifier, tc.input, got, tc.want)
			}
		})
	}
}

func TestTruncateRejectsInvalidMax(t *testing.T) {
	t.Parallel()

	for _, params := range []map[string]string{nil, {"max": ""}, {"max": "ten"}, {"max": "-1"}} {
		got, err := applyNamed(t, "truncate", params, "abc")
		if err == nil || !errors.Is(err, modifiers.ErrInvalidParameter) {
			t.Fatalf("params %v: got %v, want ErrInvalidParameter", params, err)
		}

		if got != "abc" {
			t.Fatalf("params %v: value changed to %q", params, got)
		}
	}
}
