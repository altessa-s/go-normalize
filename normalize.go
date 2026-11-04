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

// builtins is the registry Normalize resolves modifiers in. It is a private
// clone of the built-ins that nothing registers into.
var builtins = modifiers.DefaultRegistry()

// Normalize applies the built-in modifiers to the fields of the struct value
// points to, as declared by their normalize tags. Use NormalizeWithRegistry
// for modifiers outside the built-ins.
func Normalize(value any) error { return NormalizeWithRegistry(value, builtins) }

// ModifierSource resolves the modifier names used in normalize tags.
// *modifiers.Registry implements it.
type ModifierSource interface {
	Get(name string) (modifiers.Modifier, bool)
}

// NormalizeWithRegistry is Normalize with the modifiers that source resolves,
// typically a *modifiers.Registry built from modifiers.DefaultRegistry and the
// optional modules. A nil source resolves no modifiers.
func NormalizeWithRegistry(value any, source ModifierSource) error {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return ErrInvalidInput
	}

	if source == nil {
		source = (*modifiers.Registry)(nil)
	}

	w := walker{source: source}

	return resolvePath(w.walk(v.Elem()))
}
