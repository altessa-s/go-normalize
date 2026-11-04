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
// Registry lookups take no lock; Register publishes a new copy of the table,
// so modifiers are meant to be registered during setup.
//
//	registry := modifiers.NewRegistry()
//	registry.Register("slug", slug)
//	err := normalize.NormalizeWithRegistry(&value, registry)
package modifiers
