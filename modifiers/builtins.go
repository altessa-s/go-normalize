// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"reflect"
	"strings"
	"unicode"
)

func stringModifier(fn func(string) string) Modifier {
	return func(v reflect.Value, _ map[string]string) ModifierResult {
		return ApplyStringModifier(v, func(s string) (string, bool) { n := fn(s); return n, n != s })
	}
}

// ApplyStringModifier transforms string and *string values.
func ApplyStringModifier(v reflect.Value, fn func(string) (string, bool)) ModifierResult {
	if !v.IsValid() {
		return ModifierResult{}
	}
	if v.Kind() == reflect.Pointer && !v.IsNil() && v.Type().Elem().Kind() == reflect.String && !v.CanSet() {
		n, changed := fn(v.Elem().String())
		if changed {
			x := reflect.New(v.Type().Elem())
			x.Elem().SetString(n)
			return ModifierResult{Value: x}
		}
		return ModifierResult{Value: v}
	}
	leaf := v
	for leaf.Kind() == reflect.Pointer {
		if leaf.IsNil() {
			return ModifierResult{Value: v}
		}
		leaf = leaf.Elem()
	}
	if leaf.Kind() == reflect.String {
		n, c := fn(leaf.String())
		if c {
			if leaf.CanSet() {
				leaf.SetString(n)
			} else {
				x := reflect.New(leaf.Type()).Elem()
				x.SetString(n)
				return ModifierResult{Value: x}
			}
		}
		return ModifierResult{Value: v}
	}
	return ModifierResult{Value: v}
}

// NilOnEmpty sets empty *string fields to nil.
func NilOnEmpty(v reflect.Value, _ map[string]string) ModifierResult {
	if !v.IsValid() {
		return ModifierResult{Value: v}
	}
	leaf := v
	for leaf.Kind() == reflect.Pointer {
		if leaf.IsNil() {
			return ModifierResult{Value: v}
		}
		if leaf.Type().Elem().Kind() == reflect.String {
			if leaf.Elem().Len() == 0 && leaf.CanSet() {
				leaf.Set(reflect.Zero(leaf.Type()))
			}
			return ModifierResult{Value: v}
		}
		leaf = leaf.Elem()
	}
	return ModifierResult{Value: v}
}

// RemoveEmptyElements removes blank strings and nil values from string slices.
func RemoveEmptyElements(v reflect.Value, _ map[string]string) ModifierResult {
	if !v.IsValid() || v.Kind() != reflect.Slice || !v.CanSet() {
		return ModifierResult{Value: v}
	}
	e := v.Type().Elem()
	if e.Kind() != reflect.String && (e.Kind() != reflect.Pointer || e.Elem().Kind() != reflect.String) {
		return ModifierResult{Value: v}
	}
	first := 0
	for first < v.Len() && keepElement(v.Index(first), e.Kind() == reflect.Pointer) {
		first++
	}
	if first == v.Len() {
		return ModifierResult{Value: v}
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
	return ModifierResult{Value: v}
}

func keepElement(v reflect.Value, pointer bool) bool {
	if pointer {
		return !v.IsNil() && strings.TrimSpace(v.Elem().String()) != ""
	}
	return strings.TrimSpace(v.String()) != ""
}

// RemoveBadSymbols removes control and directional formatting characters.
func RemoveBadSymbols(v reflect.Value, _ map[string]string) ModifierResult {
	return ApplyStringModifier(v, func(s string) (string, bool) {
		n := strings.Map(func(r rune) rune {
			if isBadSymbol(r) {
				return -1
			}
			return r
		}, s)
		if n == "" && s != "" {
			return s, false
		}
		return n, n != s
	})
}

func isBadSymbol(r rune) bool {
	if unicode.IsControl(r) {
		return true
	}
	switch r {
	case '\u0340', '\u0341', '\u17a3', '\u17d3',
		'\u2028', '\u2029', '\u202a', '\u202b', '\u202c', '\u202d', '\u202e',
		'\u206a', '\u206b', '\u206c', '\u206d', '\u206e', '\u206f',
		'\ufff9', '\ufffa', '\ufffb', '\ufeff', '\ufffc':
		return true
	}
	return r >= 0x1d173 && r <= 0x1d17a || r >= 0xe0000 && r <= 0xe007f
}
