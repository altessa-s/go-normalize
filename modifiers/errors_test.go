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

func TestErrorFormatsContextAndUnwrapsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("invalid value")

	cases := []struct {
		name string
		err  modifiers.Error
		want string
	}{
		{name: "field fallback", err: modifiers.Error{Field: "Name", Modifier: "trim"}, want: "trim' failed at 'Name'"},
		{name: "cause", err: modifiers.Error{Path: "Contact.Name", Modifier: "trim", Err: cause}, want: "trim' failed at 'Contact.Name': invalid value"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if !strings.Contains(got, tc.want) {
				t.Fatalf("Error() = %q, want it to contain %q", got, tc.want)
			}

			if tc.err.Err != nil && !errors.Is(&tc.err, cause) {
				t.Fatalf("Unwrap() did not preserve cause: %v", tc.err.Unwrap())
			}

			if tc.err.Err == nil && tc.err.Unwrap() != nil {
				t.Fatalf("Unwrap() = %v, want nil", tc.err.Unwrap())
			}
		})
	}
}

func TestDefaultRegistryReturnsIndependentClones(t *testing.T) {
	t.Parallel()

	modifier := modifiers.Modifier(func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Value: v}
	})

	mine := modifiers.DefaultRegistry()
	if err := mine.Register("slug", modifier); err != nil {
		t.Fatal(err)
	}

	if err := mine.Register("trim", modifier); err != nil {
		t.Fatal(err)
	}

	if _, ok := mine.Get("slug"); !ok {
		t.Fatal("the clone lost its own registration")
	}

	other := modifiers.DefaultRegistry()
	if _, ok := other.Get("slug"); ok {
		t.Fatal("a registration leaked into another default registry")
	}

	trim, ok := other.Get("trim")
	if !ok || reflect.ValueOf(trim).Pointer() == reflect.ValueOf(modifier).Pointer() {
		t.Fatal("replacing a built-in in one clone changed another")
	}
}

func TestCloneIsIndependent(t *testing.T) {
	t.Parallel()

	modifier := modifiers.Modifier(func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Value: v}
	})

	source := modifiers.NewRegistry()
	_ = source.Register("a", modifier)

	clone := source.Clone()
	_ = clone.Register("b", modifier)
	_ = source.Register("c", modifier)

	for name, want := range map[string]bool{"a": true, "b": true, "c": false} {
		if _, ok := clone.Get(name); ok != want {
			t.Fatalf("clone.Get(%q) = %v, want %v", name, ok, want)
		}
	}

	if _, ok := source.Get("b"); ok {
		t.Fatal("registering into the clone changed the source")
	}

	var nilRegistry *modifiers.Registry
	if empty := nilRegistry.Clone(); empty == nil {
		t.Fatal("Clone of a nil registry returned nil")
	} else if _, ok := empty.Get("a"); ok {
		t.Fatal("Clone of a nil registry is not empty")
	}
}

// derefRegistrar dereferences its receiver, so a typed nil one panics.
type derefRegistrar struct{ names []string }

func (r *derefRegistrar) Register(name string, _ modifiers.Modifier) error {
	r.names = append(r.names, name)
	return nil
}

type mapRegistrar map[string]modifiers.Modifier

func (r mapRegistrar) Register(name string, m modifiers.Modifier) error {
	r[name] = m
	return nil
}

func TestCheckRegistrar(t *testing.T) {
	t.Parallel()

	var (
		nilPointer *derefRegistrar
		nilMap     mapRegistrar
		nilReg     *modifiers.Registry
	)

	for name, r := range map[string]modifiers.Registrar{
		"nil interface": nil,
		"typed nil":     nilPointer,
		"nil map":       nilMap,
		"nil Registry":  nilReg,
	} {
		if err := modifiers.CheckRegistrar(r); !errors.Is(err, modifiers.ErrInvalidRegistration) {
			t.Fatalf("%s: CheckRegistrar = %v, want ErrInvalidRegistration", name, err)
		}
	}

	for name, r := range map[string]modifiers.Registrar{
		"pointer":  &derefRegistrar{},
		"map":      mapRegistrar{},
		"registry": modifiers.NewRegistry(),
	} {
		if err := modifiers.CheckRegistrar(r); err != nil {
			t.Fatalf("%s: CheckRegistrar = %v", name, err)
		}
	}
}

func TestRegisterRejectsInvalidRegistrations(t *testing.T) {
	t.Parallel()

	modifier := modifiers.Modifier(func(v reflect.Value, _ map[string]string) modifiers.Result {
		return modifiers.Result{Value: v}
	})

	var nilRegistry *modifiers.Registry

	cases := []struct {
		name     string
		registry *modifiers.Registry
		key      string
		modifier modifiers.Modifier
	}{
		{name: "nil registry", registry: nilRegistry, key: "x", modifier: modifier},
		{name: "empty name", registry: modifiers.NewRegistry(), key: "", modifier: modifier},
		{name: "nil modifier", registry: modifiers.NewRegistry(), key: "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.registry.Register(tc.key, tc.modifier); !errors.Is(err, modifiers.ErrInvalidRegistration) {
				t.Fatalf("Register = %v, want ErrInvalidRegistration", err)
			}

			if _, ok := tc.registry.Get(tc.key); ok {
				t.Fatal("an invalid registration was stored")
			}
		})
	}
}
