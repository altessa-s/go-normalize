// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/altessa-s/go-normalize/modifiers"
)

func applyFuzz(t *testing.T, m modifiers.Modifier, params map[string]string, s string) string {
	t.Helper()

	if out := m(reflect.ValueOf(&s).Elem(), params); out.Err != nil {
		t.Fatal(out.Err)
	}

	return s
}

func FuzzStringModifiers(f *testing.F) {
	f.Add("  John.Doe@Example.COM ", 3)
	f.Add("a\t\u00a0 b\xff", 1)
	f.Add("+7 (916) 123-45-67", 0)
	f.Add("\u0430\U0001f600b", 2)
	f.Add(" HTTPS://User@Example.COM:443/a?b=C#D ", 0)
	f.Add("http://[::1]:80", 0)
	f.Add("\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f\U0001f3f4\U000e0031\U000e007f", 0)

	r := modifiers.DefaultRegistry()
	get := func(name string) modifiers.Modifier {
		m, ok := r.Get(name)
		if !ok {
			f.Fatalf("%s is not registered", name)
		}

		return m
	}
	collapse, truncate, digits, email, urlModifier := get("collapse_spaces"), get("truncate"), get("digits_only"), get("email"), get("url")
	removeBad := get("remove_bad_symbols")

	f.Fuzz(func(t *testing.T, s string, limit int) {
		valid := utf8.ValidString(s)

		collapsed := applyFuzz(t, collapse, nil, s)
		if strings.Contains(collapsed, "  ") || (valid && !utf8.ValidString(collapsed)) {
			t.Fatalf("collapse_spaces(%q) = %q", s, collapsed)
		}

		if again := applyFuzz(t, collapse, nil, collapsed); again != collapsed {
			t.Fatalf("collapse_spaces is not idempotent: %q -> %q", collapsed, again)
		}

		cleaned := applyFuzz(t, removeBad, nil, s)
		if again := applyFuzz(t, removeBad, nil, cleaned); again != cleaned || !utf8.ValidString(cleaned) {
			t.Fatalf("remove_bad_symbols(%+q) = %+q, again %+q", s, cleaned, again)
		}

		if strings.ContainsFunc(cleaned, func(r rune) bool { return unicode.In(r, unicode.Bidi_Control, unicode.Cc) }) {
			t.Fatalf("remove_bad_symbols(%+q) = %+q keeps a control", s, cleaned)
		}

		limit = min(max(limit, 0), 64)

		truncated := applyFuzz(t, truncate, map[string]string{"max": strconv.Itoa(limit)}, s)
		if !strings.HasPrefix(s, truncated) || utf8.RuneCountInString(truncated) > limit || (valid && !utf8.ValidString(truncated)) {
			t.Fatalf("truncate(%q, %d) = %q", s, limit, truncated)
		}

		digitsOnly := applyFuzz(t, digits, nil, s)
		if strings.TrimLeft(digitsOnly, "0123456789") != "" {
			t.Fatalf("digits_only(%q) = %q", s, digitsOnly)
		}

		normalizedURL := applyFuzz(t, urlModifier, nil, s)
		if again := applyFuzz(t, urlModifier, nil, normalizedURL); again != normalizedURL {
			t.Fatalf("url is not idempotent: %q -> %q -> %q", s, normalizedURL, again)
		}

		normalized := applyFuzz(t, email, nil, s)
		if again := applyFuzz(t, email, nil, normalized); again != normalized {
			t.Fatalf("email is not idempotent: %q -> %q", normalized, again)
		}
	})
}
