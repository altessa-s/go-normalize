// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package normalize applies field modifiers declared by normalize struct tags.
//
// A tag lists modifier clauses separated by commas, applied in order. A clause
// may take parameters, as in phone(region=US). The tag "-" skips a field and
// TagValueCustom leaves it to the parent's CustomNormalizer.
//
//	type Contact struct {
//		Name  string  `normalize:"trim,lowercase"`
//		Email *string `normalize:"trim,lowercase,nil_on_empty"`
//	}
//
//	err := normalize.Normalize(&contact)
//
// # Traversal
//
// Normalize visits exported fields and follows pointers, slices, arrays, maps
// and interfaces, stopping at cycles. It also descends into unexported
// embedded structs and pointers to them, whose exported fields are promoted;
// errors name such fields as if they belonged to the outer struct. Map values are normalized as copies and written back only when they
// change. A struct whose pointer implements CustomNormalizer has its Normalize
// method called after its tagged fields. The fields to visit and their parsed
// clauses are computed once per struct type.
//
// # Modifiers
//
// Normalize resolves modifiers in modifiers.DefaultRegistry.
// NormalizeWithRegistry takes any ModifierSource, such as a
// *modifiers.Registry extended with the phone module.
//
// # Errors
//
// ErrInvalidInput, ErrModifierNotFound and ErrNonReflexiveMapKey report
// misuse; modifier failures are *modifiers.Error values carrying the
// field path, including a result of the wrong type, reported as
// ErrResultType. Map keys appear in paths as {key}.
package normalize
