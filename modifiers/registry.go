// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
)

// ErrInvalidRegistration reports a Register call on a nil registry, with an
// empty name or with a nil modifier.
var ErrInvalidRegistration = errors.New("modifiers: invalid registration")

// Registrar accepts modifier registrations. *Registry implements it, and the
// Register functions of the phone, text and idna modules take one.
type Registrar interface {
	Register(name string, modifier Modifier) error
}

// Registry stores modifier functions and is safe for concurrent use. Lookups
// never block: Register publishes a new copy of the table, so registration is
// meant for setup rather than hot paths. The zero value is an empty registry.
type Registry struct {
	mu    sync.Mutex
	items atomic.Pointer[map[string]Modifier]
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{} }

// Register adds a modifier under name, replacing any modifier already
// registered under it. A nil registry, an empty name or a nil modifier fails
// with ErrInvalidRegistration.
func (r *Registry) Register(name string, modifier Modifier) error {
	switch {
	case r == nil:
		return fmt.Errorf("%w: nil registry", ErrInvalidRegistration)
	case name == "":
		return fmt.Errorf("%w: empty name", ErrInvalidRegistration)
	case modifier == nil:
		return fmt.Errorf("%w: nil modifier for %q", ErrInvalidRegistration, name)
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

	return nil
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

// Clone returns a new registry holding the modifiers of r. Registering into
// either registry afterwards does not affect the other. Cloning a nil registry
// returns an empty one.
func (r *Registry) Clone() *Registry {
	c := NewRegistry()
	if r == nil {
		return c
	}
	// Register never changes a published table, so the clone shares it until
	// either side registers again.
	c.items.Store(r.items.Load())

	return c
}

// builtins holds the built-in modifiers. Nothing registers into it after
// initialization; DefaultRegistry hands out clones.
var builtins = newBuiltins()

// DefaultRegistry returns a new registry holding the built-in modifiers. Each
// call returns a separate registry owned by the caller, so registering into
// it, for example with phone.Register, affects no other registry. The call
// does not copy the table and is cheap.
func DefaultRegistry() *Registry { return builtins.Clone() }

func newBuiltins() *Registry {
	r := NewRegistry()
	for name, m := range map[string]Modifier{
		"trim":                  stringModifier(strings.TrimSpace),
		"lowercase":             stringModifier(strings.ToLower),
		"uppercase":             stringModifier(strings.ToUpper),
		"nil_on_empty":          NilOnEmpty,
		"remove_bad_symbols":    RemoveBadSymbols,
		"remove_empty_elements": RemoveEmptyElements,
		"collapse_spaces":       CollapseSpaces,
		"truncate":              Truncate,
		"digits_only":           DigitsOnly,
		"email":                 Email,
		"nil_on_zero":           NilOnZero,
		"empty_on_nil":          EmptyOnNil,
		"dedupe":                Dedupe,
		"utc":                   UTC,
		"truncate_time":         TruncateTime,
		"clamp":                 Clamp,
		"url":                   URL,
	} {
		// Every name is set and every modifier non-nil, so Register cannot fail.
		_ = r.Register(name, m)
	}

	return r
}
