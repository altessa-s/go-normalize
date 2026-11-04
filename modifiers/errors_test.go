// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

func TestModifierErrorFormatsContextAndUnwrapsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("invalid value")

	cases := []struct {
		name string
		err  modifiers.ModifierError
		want string
	}{
		{name: "field fallback", err: modifiers.ModifierError{FieldName: "Name", ModifierName: "trim"}, want: "trim' failed at 'Name'"},
		{name: "cause", err: modifiers.ModifierError{FieldPath: "Contact.Name", ModifierName: "trim", Cause: cause}, want: "trim' failed at 'Contact.Name': invalid value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if !strings.Contains(got, tc.want) {
				t.Fatalf("Error() = %q, want it to contain %q", got, tc.want)
			}

			if tc.err.Cause != nil && !errors.Is(&tc.err, cause) {
				t.Fatalf("Unwrap() did not preserve cause: %v", tc.err.Unwrap())
			}

			if tc.err.Cause == nil && tc.err.Unwrap() != nil {
				t.Fatalf("Unwrap() = %v, want nil", tc.err.Unwrap())
			}
		})
	}
}

func TestDefaultRegistryCompatibilityHelpers(t *testing.T) {
	name := "test_default_registry_helper"
	if _, ok := modifiers.GetModifier(name); ok {
		t.Fatal("test modifier unexpectedly exists")
	}

	modifier := modifiers.Modifier(func(v reflect.Value, _ map[string]string) modifiers.ModifierResult {
		return modifiers.ModifierResult{Value: v}
	})
	modifiers.RegisterModifier(name, modifier)

	got, ok := modifiers.GetModifier(name)
	if !ok || reflect.ValueOf(got).Pointer() != reflect.ValueOf(modifier).Pointer() {
		t.Fatal("compatibility helpers did not expose the registered modifier")
	}
}
