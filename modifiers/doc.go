// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package modifiers provides the modifier types, a concurrency-safe registry
// and the built-in modifiers used by normalize tags.
//
// # Built-ins
//
// DefaultRegistry holds trim, lowercase, uppercase, nil_on_empty,
// remove_bad_symbols, remove_empty_elements, collapse_spaces, truncate(max=N),
// digits_only, email, nil_on_zero, empty_on_nil, dedupe, utc,
// truncate_time(unit=U), clamp(min=A;max=B) and url.
//
// # Registry
//
// DefaultRegistry returns a new registry holding the built-ins; every call
// returns a separate registry, so there is no process-wide state to change.
// NewRegistry returns an empty one and Clone copies any registry. Registry
// lookups take no lock; Register publishes a new copy of the table, so
// modifiers are meant to be registered during setup.
//
//	registry := modifiers.DefaultRegistry()
//	if err := registry.Register("slug", slug); err != nil {
//		return err
//	}
//	err := normalize.NormalizeWithRegistry(&value, registry)
package modifiers
