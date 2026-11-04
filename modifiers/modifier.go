// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"errors"
	"fmt"
	"reflect"
)

// ErrInvalidParameter is the cause of an Error for a tag parameter the
// modifier cannot use.
var ErrInvalidParameter = errors.New("modifiers: invalid parameter")

// Modifier transforms a reflected field value. params holds the clause
// parameters; it is a fresh map on every call, so a modifier may change it.
type Modifier func(v reflect.Value, params map[string]string) Result

// Result is the outcome of a modifier call.
type Result struct {
	// Value is the value to store in the field. The zero Value means the
	// field is left as it is. A valid Value must be assignable to the field;
	// normalize reports any other type as an error.
	Value reflect.Value
	// Err reports why the modifier failed; normalize wraps it in an *Error
	// that adds the field and the modifier name. Value is ignored when Err is
	// set.
	Err error
}

// Error describes a modifier failure at a struct field. normalize returns it
// for every non-nil Result.Err.
type Error struct {
	// Modifier is the name of the modifier as written in the tag.
	Modifier string
	// Field is the name of the struct field that held the value.
	Field string
	// Path locates the field from the value passed to normalize, as in
	// Items[1].Name; map keys appear as {key}. It may be empty, in which case
	// Error reports Field.
	Path string
	// Err is the error the modifier returned.
	Err error
}

// Error describes the failure with the modifier name and field path.
func (e *Error) Error() string {
	path := e.Path
	if path == "" {
		path = e.Field
	}

	if e.Err == nil {
		return "normalize: modifier '" + e.Modifier + "' failed at '" + path + "'"
	}

	return "normalize: modifier '" + e.Modifier + "' failed at '" + path + "': " + e.Err.Error()
}

// Unwrap returns the error the modifier returned.
func (e *Error) Unwrap() error { return e.Err }

// Clause is a parsed tag clause: a modifier name and its parameters.
type Clause struct {
	// Name is the modifier name, such as trim or phone.
	Name string
	// Params maps parameter names to their values; it is nil for a clause
	// written without parentheses.
	Params map[string]string
}

// invalidParameter reports a tag parameter the modifier cannot use; v is left unchanged.
func invalidParameter(v reflect.Value, format string, args ...any) Result {
	return Result{Value: v, Err: fmt.Errorf("%w: "+format, append([]any{ErrInvalidParameter}, args...)...)}
}
