// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"maps"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry stores modifier functions and is safe for concurrent use. Lookups
// never block: Register publishes a new copy of the table, so registration is
// meant for setup rather than hot paths. The zero value is an empty registry.
type Registry struct {
	mu    sync.Mutex
	items atomic.Pointer[map[string]Modifier]
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{} }

// Register adds or replaces a modifier.
func (r *Registry) Register(name string, modifier Modifier) {
	if r == nil || name == "" || modifier == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var items map[string]Modifier
	if current := r.items.Load(); current != nil {
		items = make(map[string]Modifier, len(*current)+1)
		maps.Copy(items, *current)
	} else {
		items = make(map[string]Modifier, 1)
	}
	items[name] = modifier
	r.items.Store(&items)
}

// Get returns a registered modifier.
func (r *Registry) Get(name string) (Modifier, bool) {
	if r == nil {
		return nil, false
	}
	items := r.items.Load()
	if items == nil {
		return nil, false
	}
	m, ok := (*items)[name]
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
	r.Register("collapse_spaces", CollapseSpaces)
	r.Register("truncate", Truncate)
	r.Register("digits_only", DigitsOnly)
	r.Register("email", Email)
	r.Register("nil_on_zero", NilOnZero)
	r.Register("empty_on_nil", EmptyOnNil)
	r.Register("dedupe", Dedupe)
	r.Register("utc", UTC)
	r.Register("truncate_time", TruncateTime)
	r.Register("clamp", Clamp)
	r.Register("url", URL)
	return r
}
