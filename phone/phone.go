// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone

import (
	"cmp"
	"errors"
	"reflect"
	"strings"

	"github.com/nyaruka/phonenumbers"

	"github.com/altessa-s/go-normalize/modifiers"
)

// ErrInvalidPhone identifies an invalid phone number.
var ErrInvalidPhone = errors.New("phone number is invalid")

// defaultRegion applies when the tag names no region.
const defaultRegion = "RU"

// separators are the formatting characters removed before parsing.
var separators = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "", "_", "")

// Registrar accepts modifier registrations. *modifiers.Registry implements it.
type Registrar interface {
	Register(name string, modifier modifiers.Modifier)
}

// Register adds the phone modifier to registry. Registration is explicit and has no package side effect.
// A nil registry is ignored.
func Register(registry Registrar) {
	if registry == nil {
		return
	}
	registry.Register("phone", NormalizePhone)
}

// NormalizePhone converts a valid phone number to E.164 form. Region defaults to RU.
func NormalizePhone(v reflect.Value, params map[string]string) modifiers.ModifierResult {
	if !v.IsValid() {
		return modifiers.ModifierResult{Value: v}
	}
	var s string
	ptr := false
	switch v.Kind() {
	case reflect.String:
		s = v.String()
	case reflect.Pointer:
		if v.IsNil() || v.Type().Elem().Kind() != reflect.String {
			return modifiers.ModifierResult{Value: v}
		}
		s = v.Elem().String()
		ptr = true
	default:
		return modifiers.ModifierResult{Value: v}
	}
	if s == "" {
		return modifiers.ModifierResult{Value: v}
	}
	region := cmp.Or(strings.ToUpper(params["region"]), defaultRegion)
	key := cacheKey{region: region, number: s}
	cacheable := len(region)+len(s) <= maxPhoneCacheKeyLength
	if cacheable {
		if got, ok := phoneCache.get(key); ok {
			if got.err {
				return fail(v)
			}
			return set(v, got.value, ptr)
		}
	}
	n, err := phonenumbers.Parse(separators.Replace(s), region)
	if err != nil || !phonenumbers.IsValidNumber(n) {
		if cacheable {
			phoneCache.put(key, cacheEntry{err: true})
		}
		return fail(v)
	}
	out := phonenumbers.Format(n, phonenumbers.E164)
	if cacheable {
		phoneCache.put(key, cacheEntry{value: out})
	}
	return set(v, out, ptr)
}

func fail(v reflect.Value) modifiers.ModifierResult {
	return modifiers.ModifierResult{Value: v, Error: &modifiers.ModifierError{ModifierName: "phone", Cause: ErrInvalidPhone}}
}
func set(v reflect.Value, s string, ptr bool) modifiers.ModifierResult {
	if ptr {
		x := reflect.New(v.Type().Elem())
		x.Elem().SetString(s)
		if v.CanSet() {
			v.Set(x)
			return modifiers.ModifierResult{Value: v}
		}
		return modifiers.ModifierResult{Value: x}
	}
	if s == v.String() {
		return modifiers.ModifierResult{Value: v}
	}
	if v.CanSet() {
		v.SetString(s)
		return modifiers.ModifierResult{Value: v}
	}
	x := reflect.New(v.Type()).Elem()
	x.SetString(s)
	return modifiers.ModifierResult{Value: x}
}
