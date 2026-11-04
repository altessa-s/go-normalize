// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func stringModifier(fn func(string) string) Modifier {
	return func(v reflect.Value, _ map[string]string) Result {
		return ApplyString(v, func(s string) (string, bool) { n := fn(s); return n, n != s })
	}
}

// NilOnEmpty sets empty *string fields to nil. In a pointer chain the innermost
// pointer is cleared.
func NilOnEmpty(v reflect.Value, _ map[string]string) Result {
	return clearInnermost(v, func(x reflect.Value) bool { return x.Kind() == reflect.String && x.Len() == 0 })
}

// clearInnermost sets the innermost pointer of the chain v to nil when empty
// reports the value it points to. A chain that cannot be set is copied first.
func clearInnermost(v reflect.Value, empty func(reflect.Value) bool) Result {
	if !v.IsValid() || v.Kind() != reflect.Pointer {
		return Result{Value: v}
	}

	if !v.CanSet() {
		root, _, _, ok := copyChain(v, false)
		if ok && clearPointer(root, empty) {
			return Result{Value: root}
		}

		return Result{Value: v}
	}

	clearPointer(v, empty)

	return Result{Value: v}
}

func clearPointer(p reflect.Value, empty func(reflect.Value) bool) bool {
	for !p.IsNil() && p.Elem().Kind() == reflect.Pointer {
		p = p.Elem()
	}

	if p.IsNil() || !empty(p.Elem()) {
		return false
	}

	p.Set(reflect.Zero(p.Type()))

	return true
}

// RemoveEmptyElements removes blank strings and nil values from string slices.
func RemoveEmptyElements(v reflect.Value, _ map[string]string) Result {
	if !v.IsValid() || v.Kind() != reflect.Slice || !v.CanSet() {
		return Result{Value: v}
	}

	e := v.Type().Elem()
	if e.Kind() != reflect.String && (e.Kind() != reflect.Pointer || e.Elem().Kind() != reflect.String) {
		return Result{Value: v}
	}

	first := 0
	for first < v.Len() && keepElement(v.Index(first), e.Kind() == reflect.Pointer) {
		first++
	}

	if first == v.Len() {
		return Result{Value: v}
	}

	n := reflect.MakeSlice(v.Type(), first, v.Len()-1)
	reflect.Copy(n, v.Slice(0, first))

	for i := first + 1; i < v.Len(); i++ {
		x := v.Index(i)
		if keepElement(x, e.Kind() == reflect.Pointer) {
			n = reflect.Append(n, x)
		}
	}

	v.Set(n)

	return Result{Value: v}
}

func keepElement(v reflect.Value, pointer bool) bool {
	if pointer {
		return !v.IsNil() && strings.TrimSpace(v.Elem().String()) != ""
	}

	return strings.TrimSpace(v.String()) != ""
}

// CollapseSpaces replaces every run of white space, as defined by
// unicode.IsSpace, with a single ASCII space. It does not trim; combine it with
// trim for that. Bytes that are not valid UTF-8 are kept as they are.
func CollapseSpaces(v reflect.Value, _ map[string]string) Result {
	return ApplyString(v, collapseSpaces)
}

func collapseSpaces(s string) (string, bool) {
	if !needsCollapse(s) {
		return s, false
	}

	var b strings.Builder
	b.Grow(len(s))

	inSpace := false

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
			}

			inSpace = true
		} else {
			b.WriteString(s[i : i+size])

			inSpace = false
		}

		i += size
	}

	return b.String(), true
}

// needsCollapse reports whether s has white space other than single ASCII spaces.
func needsCollapse(s string) bool {
	inSpace := false

	for _, r := range s {
		space := unicode.IsSpace(r)
		if space && (inSpace || r != ' ') {
			return true
		}

		inSpace = space
	}

	return false
}

// Truncate shortens strings to at most max runes, given as the max parameter,
// and never cuts a multi-byte character. A missing, non-numeric or negative max
// fails with ErrInvalidParameter.
func Truncate(v reflect.Value, params map[string]string) Result {
	raw, ok := params["max"]

	limit, err := strconv.Atoi(raw)
	if !ok || err != nil || limit < 0 {
		return invalidParameter(v, "max must be a non-negative integer, got %q", raw)
	}

	return ApplyString(v, func(s string) (string, bool) { return truncateRunes(s, limit) })
}

func truncateRunes(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}

	count := 0
	for i := range s {
		if count == limit {
			return s[:i], true
		}

		count++
	}

	return s, false
}

// DigitsOnly keeps the ASCII digits 0-9 and removes every other character.
func DigitsOnly(v reflect.Value, _ map[string]string) Result {
	return ApplyString(v, func(s string) (string, bool) {
		n := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}

			return -1
		}, s)

		return n, n != s
	})
}

// Email trims an address and lower-cases its domain, the part after the last
// '@'. The local part keeps its case, which RFC 5321 lets servers distinguish.
// It does not validate the address; a value without '@' is only trimmed.
func Email(v reflect.Value, _ map[string]string) Result {
	return ApplyString(v, normalizeEmail)
}

func normalizeEmail(s string) (string, bool) {
	n := strings.TrimSpace(s)
	if local, domain, ok := strings.CutLast(n, "@"); ok {
		if lower := strings.ToLower(domain); lower != domain {
			n = local + "@" + lower
		}
	}

	return n, n != s
}

// NilOnZero sets a pointer to nil when it points to the zero value of its type,
// such as 0, false, "" or an empty struct. In a pointer chain the innermost
// pointer is cleared. It generalizes nil_on_empty to every pointer type.
func NilOnZero(v reflect.Value, _ map[string]string) Result {
	return clearInnermost(v, reflect.Value.IsZero)
}

// EmptyOnNil points nil pointers at new zero values, through the whole pointer
// chain, so that a *string field always holds a string. It is the inverse of
// nil_on_zero. A chain that cannot be set is copied, never written.
func EmptyOnNil(v reflect.Value, _ map[string]string) Result {
	if !v.IsValid() || v.Kind() != reflect.Pointer {
		return Result{Value: v}
	}

	if !v.CanSet() {
		if root, _, filled, _ := copyChain(v, true); filled {
			return Result{Value: root}
		}

		return Result{Value: v}
	}

	for p := v; p.Kind() == reflect.Pointer; p = p.Elem() {
		if p.IsNil() {
			p.Set(reflect.New(p.Type().Elem()))
		}
	}

	return Result{Value: v}
}

// Dedupe removes repeated elements from slices of strings, numbers and
// booleans, named types included, and keeps the first occurrence of each.
// Other element types are left unchanged: pointers and interfaces would
// compare by identity rather than by the values they refer to.
func Dedupe(v reflect.Value, _ map[string]string) Result {
	if !v.IsValid() || v.Kind() != reflect.Slice || !v.CanSet() || v.Len() < 2 {
		return Result{Value: v}
	}

	switch kind := v.Type().Elem().Kind(); {
	case kind == reflect.String:
		dedupe(v, reflect.Value.String)
	case isBasic(kind):
		dedupe(v, reflect.Value.Interface)
	}

	return Result{Value: v}
}

// dedupe replaces v with a copy without repeated keys. A slice without
// repeats keeps its backing array.
func dedupe[K comparable](v reflect.Value, key func(reflect.Value) K) {
	seen := make(map[K]struct{}, v.Len())
	first := -1

	for i := range v.Len() {
		k := key(v.Index(i))
		if _, ok := seen[k]; ok {
			first = i
			break
		}

		seen[k] = struct{}{}
	}

	if first < 0 {
		return
	}

	n := reflect.MakeSlice(v.Type(), first, v.Len()-1)
	reflect.Copy(n, v.Slice(0, first))

	for i := first + 1; i < v.Len(); i++ {
		x := v.Index(i)

		k := key(x)
		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		n = reflect.Append(n, x)
	}

	v.Set(n)
}
