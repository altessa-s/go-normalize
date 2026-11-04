// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"errors"
	"fmt"
	"reflect"
)

// ErrInvalidParameter is the cause of a ModifierError for a tag parameter the
// modifier cannot use.
var ErrInvalidParameter = errors.New("modifiers: invalid parameter")

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

// invalidParameter reports a tag parameter the modifier cannot use; v is left unchanged.
func invalidParameter(v reflect.Value, format string, args ...any) ModifierResult {
	return ModifierResult{Value: v, Error: &ModifierError{Cause: fmt.Errorf("%w: "+format, append([]any{ErrInvalidParameter}, args...)...)}}
}
