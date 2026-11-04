// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"maps"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// Modifier transforms a reflected field value.
type Modifier func(reflect.Value, map[string]string) ModifierResult

// ModifierResult is the result of a modifier call.
type ModifierResult struct {
	Value reflect.Value
	Error *ModifierError
}

// ModifierError describes a modifier failure.
type ModifierError struct {
	FieldName, FieldPath, ModifierName, OriginalValue string
	Cause                                             error
}

func (e *ModifierError) Error() string {
	path := e.FieldPath
	if path == "" {
		path = e.FieldName
	}
	if e.Cause == nil {
		return "normalize: modifier '" + e.ModifierName + "' failed at '" + path + "'"
	}
	return "normalize: modifier '" + e.ModifierName + "' failed at '" + path + "': " + e.Cause.Error()
}
func (e *ModifierError) Unwrap() error { return e.Cause }

// ModifierParams is a parsed modifier name and parameter map.
type ModifierParams struct {
	Name   string
	Params map[string]string
}

// Registry stores modifier functions and is safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	items map[string]Modifier
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{items: make(map[string]Modifier)} }

// Register adds or replaces a modifier.
func (r *Registry) Register(name string, modifier Modifier) {
	if r == nil || name == "" || modifier == nil {
		return
	}
	r.mu.Lock()
	r.items[name] = modifier
	r.mu.Unlock()
}

// Get returns a registered modifier.
func (r *Registry) Get(name string) (Modifier, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.items[name]
	return m, ok
}

// RegisterModifier registers a modifier in the compatibility default registry.
func RegisterModifier(name string, modifier Modifier) { DefaultRegistry().Register(name, modifier) }

// GetModifier looks up a modifier in the default registry.
func GetModifier(name string) (Modifier, bool) { return DefaultRegistry().Get(name) }

var defaultRegistry = buildDefaultRegistry()

// DefaultRegistry returns the built-in modifier registry.
func DefaultRegistry() *Registry { return defaultRegistry }
func buildDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register("trim", stringModifier(strings.TrimSpace))
	r.Register("lowercase", stringModifier(strings.ToLower))
	r.Register("uppercase", stringModifier(strings.ToUpper))
	r.Register("nil_on_empty", NilOnEmpty)
	r.Register("remove_bad_symbols", RemoveBadSymbols)
	r.Register("remove_empty_elements", RemoveEmptyElements)
	return r
}
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

// Parse parses a modifier clause such as trim or phone(region=US).
const (
	modifierParseCacheLimit = 256
	maxCachedModifierLength = 1024
)

type modifierParseCache struct {
	mu     sync.Mutex
	values map[string]ModifierParams
	keys   []string
}

var parsedModifiers = &modifierParseCache{values: make(map[string]ModifierParams, modifierParseCacheLimit)}

// Parse parses a modifier clause and caches bounded-size clauses safely.
func Parse(s string) ModifierParams {
	if len(s) <= maxCachedModifierLength {
		parsedModifiers.mu.Lock()
		if value, ok := parsedModifiers.values[s]; ok {
			parsedModifiers.mu.Unlock()
			return cloneParams(value)
		}
		parsedModifiers.mu.Unlock()
	}
	value := parseUncached(s)
	if len(s) <= maxCachedModifierLength {
		parsedModifiers.mu.Lock()
		if _, ok := parsedModifiers.values[s]; !ok {
			if len(parsedModifiers.keys) == modifierParseCacheLimit {
				delete(parsedModifiers.values, parsedModifiers.keys[0])
				parsedModifiers.keys = parsedModifiers.keys[1:]
			}
			parsedModifiers.keys = append(parsedModifiers.keys, s)
		}
		parsedModifiers.values[s] = cloneParams(value)
		parsedModifiers.mu.Unlock()
	}
	return value
}

func cloneParams(value ModifierParams) ModifierParams {
	if value.Params == nil {
		return value
	}
	params := make(map[string]string, len(value.Params))
	maps.Copy(params, value.Params)
	value.Params = params
	return value
}

func parseUncached(s string) ModifierParams {
	s = strings.TrimSpace(s)
	i := strings.IndexByte(s, '(')
	if i < 0 {
		return ModifierParams{Name: s}
	}
	name := strings.TrimSpace(s[:i])
	if name == "" || !strings.HasSuffix(s, ")") {
		return ModifierParams{Name: s}
	}
	p := map[string]string{}
	for part := range strings.SplitSeq(s[i+1:len(s)-1], ";") {
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			p[strings.TrimSpace(k)] = ""
		} else {
			p[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return ModifierParams{Name: name, Params: p}
}
