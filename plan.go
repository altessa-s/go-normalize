// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"reflect"
	"strings"
	"sync"

	"github.com/altessa-s/go-normalize/modifiers"
)

const tagName = "normalize"

// maxElemDepth bounds the element chain needsWalk follows; recursive
// unnamed-element types such as `type P *P` would otherwise never end.
const maxElemDepth = 16

var customNormalizerType = reflect.TypeFor[CustomNormalizer]()

// structPlan is what traversal needs to know about a struct type. It depends
// only on the type, so it is computed once and shared by every call.
type structPlan struct {
	fields []fieldPlan
	// custom reports whether *T implements CustomNormalizer.
	custom bool
}

// fieldPlan describes an exported field that has modifiers or may contain
// values with modifiers, or an unexported embedded struct whose promoted
// fields may have them. Other fields are left out of the plan.
type fieldPlan struct {
	index    int
	name     string
	clauses  []modifiers.Clause
	traverse bool
	// promoted marks an unexported embedded struct. Its fields read as fields
	// of the outer struct, so it adds no segment to error paths.
	promoted bool
	// outerCustom marks an embedded struct inside a CustomNormalizer. The
	// outer struct's Normalize is either promoted from the embedded struct or
	// overrides it, so the embedded struct's own is not called separately.
	outerCustom bool
}

var plans sync.Map // reflect.Type -> *structPlan

// planFor returns the cached plan for t. Concurrent first calls may each build
// a plan; the plans are identical, so whichever is stored last is kept.
func planFor(t reflect.Type) *structPlan {
	if cached, ok := plans.Load(t); ok {
		if plan, ok := cached.(*structPlan); ok {
			return plan
		}
	}

	plan := buildPlan(t)
	plans.Store(t, plan)

	return plan
}

func buildPlan(t reflect.Type) *structPlan {
	plan := &structPlan{custom: reflect.PointerTo(t).Implements(customNormalizerType)}
	for f := range t.Fields() {
		tag := strings.TrimSpace(f.Tag.Get(tagName))
		if tag == "-" || tag == TagValueCustom {
			continue
		}

		if !f.IsExported() {
			// An unexported embedded field is read-only, but reflect clears
			// that mark again for the exported fields of the struct it holds or
			// points to: Field drops the embedded read-only flag, also after
			// Elem of an embedded pointer. Those promoted fields are therefore
			// settable whenever the outer struct is, so traversal descends into
			// value and pointer embeds alike. The embed's own tag has no value
			// to modify.
			if f.Anonymous && embedsStruct(f.Type) {
				plan.fields = append(plan.fields, fieldPlan{index: f.Index[0], name: f.Name, traverse: true, promoted: true, outerCustom: plan.custom})
			}

			continue
		}

		field := fieldPlan{
			index:       f.Index[0],
			name:        f.Name,
			traverse:    needsWalk(f.Type),
			outerCustom: plan.custom && f.Anonymous && embedsStruct(f.Type),
		}
		for clause := range strings.SplitSeq(tag, ",") {
			if p := modifiers.Parse(clause); p.Name != "" {
				field.clauses = append(field.clauses, p)
			}
		}

		if len(field.clauses) > 0 || field.traverse {
			plan.fields = append(plan.fields, field)
		}
	}

	return plan
}

// embedsStruct reports whether an embedded field of type t is a struct or a
// pointer to one, the only embedded types with promoted fields to normalize.
func embedsStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct
}

// needsWalk reports whether a value of type t can hold a struct or an
// interface, the only places where traversal finds work.
func needsWalk(t reflect.Type) bool {
	for range maxElemDepth {
		switch t.Kind() { //nolint:exhaustive // every other kind holds no struct fields
		case reflect.Struct, reflect.Interface:
			return true
		case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			t = t.Elem()
		default:
			return false
		}
	}

	return true
}
