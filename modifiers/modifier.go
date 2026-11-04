// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import "reflect"

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
