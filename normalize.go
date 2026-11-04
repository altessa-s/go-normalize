// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"errors"
	"reflect"

	"github.com/altessa-s/go-normalize/modifiers"
)

// TagValueCustom marks fields that are handled only by their parent's CustomNormalizer.
const TagValueCustom = "custom"

// ErrInvalidInput identifies invalid Normalize inputs.
var ErrInvalidInput = errors.New("normalize: input must be a non-nil pointer to a struct")

// ErrModifierNotFound identifies an unregistered tag modifier.
var ErrModifierNotFound = errors.New("normalize: modifier not found")

// ErrNonReflexiveMapKey identifies a map value that changed under a key that
// cannot be looked up again, such as a NaN floating-point key.
var ErrNonReflexiveMapKey = errors.New("normalize: cannot update map value with non-reflexive key")

// CustomNormalizer can provide additional normalization after tagged fields.
type CustomNormalizer interface{ Normalize() error }

// Normalize applies built-in modifiers to fields tagged with normalize.
func Normalize(value any) error { return NormalizeWithRegistry(value, modifiers.DefaultRegistry()) }

// ModifierSource resolves the modifier names used in normalize tags.
// *modifiers.Registry implements it.
type ModifierSource interface {
	Get(name string) (modifiers.Modifier, bool)
}

// NormalizeWithRegistry applies modifiers from registry; it allows optional adapters such as phone.
// A nil registry resolves no modifiers.
func NormalizeWithRegistry(value any, registry ModifierSource) error {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return ErrInvalidInput
	}

	if registry == nil {
		registry = (*modifiers.Registry)(nil)
	}

	w := walker{source: registry}

	return resolvePath(w.walk(v.Elem()))
}
