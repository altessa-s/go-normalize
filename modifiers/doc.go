// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package modifiers provides the modifier types, a concurrency-safe registry
// and the built-in modifiers used by normalize tags.
//
// # Built-ins
//
// DefaultRegistry holds trim, lowercase, uppercase, nil_on_empty,
// remove_bad_symbols and remove_empty_elements.
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
