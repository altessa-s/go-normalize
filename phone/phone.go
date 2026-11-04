// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/nyaruka/phonenumbers"

	"github.com/altessa-s/go-normalize/modifiers"
)

// ErrInvalidPhone identifies a phone number that does not parse or is not
// valid. Errors do not include the number.
var ErrInvalidPhone = errors.New("phone: invalid phone number")

// ErrUnsupportedRegion identifies a region that is not an ISO 3166-1 alpha-2
// code with phone number metadata.
var ErrUnsupportedRegion = errors.New("phone: unsupported region")

// separators are the formatting characters removed before parsing.
var separators = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "", "_", "")

// Register adds the phone modifier to registry. defaultRegion is the ISO
// 3166-1 alpha-2 region, in any case, used for numbers written without a
// country code when the tag names no region; there is no implicit default.
// Registration is explicit and has no package side effect. A nil registry
// fails with modifiers.ErrInvalidRegistration and an unknown region with
// ErrUnsupportedRegion.
func Register(registry modifiers.Registrar, defaultRegion string) error {
	if registry == nil {
		return fmt.Errorf("%w: nil registry", modifiers.ErrInvalidRegistration)
	}

	m, err := New(defaultRegion)
	if err != nil {
		return err
	}

	return registry.Register("phone", m)
}

// New returns the phone modifier with defaultRegion, as Register describes.
// The modifier formats valid numbers in E.164. It applies to string values
// and pointer chains ending in one, named types included; empty strings are
// left unchanged. The tag parameter region overrides defaultRegion; a region
// without phone number metadata fails with modifiers.ErrInvalidParameter.
// A number that does not parse or is not valid fails with ErrInvalidPhone.
func New(defaultRegion string) (modifiers.Modifier, error) {
	fallback, ok := canonicalRegion(defaultRegion)
	if !ok {
		return nil, fmt.Errorf("%w: default region %q", ErrUnsupportedRegion, defaultRegion)
	}

	return func(v reflect.Value, params map[string]string) modifiers.Result {
		region := fallback
		if raw, ok := params["region"]; ok {
			if region, ok = canonicalRegion(raw); !ok {
				return modifiers.Result{Value: v, Err: fmt.Errorf("%w: %w %q", modifiers.ErrInvalidParameter, ErrUnsupportedRegion, raw)}
			}
		}

		return normalizeValue(v, region)
	}, nil
}

// canonicalRegion returns region in upper case and reports whether phone
// number metadata exists for it.
func canonicalRegion(region string) (string, bool) {
	region = strings.ToUpper(strings.TrimSpace(region))
	return region, phonenumbers.GetSupportedRegions()[region]
}

func normalizeValue(v reflect.Value, region string) modifiers.Result {
	failed := false
	out := modifiers.ApplyString(v, func(s string) (string, bool) {
		if s == "" {
			return s, false
		}

		n, ok := format(s, region)
		if !ok {
			failed = true
			return s, false
		}

		return n, n != s
	})

	if failed {
		return modifiers.Result{Value: v, Err: ErrInvalidPhone}
	}

	return out
}

// format returns s in E.164 and reports whether s is a valid number.
func format(s, region string) (string, bool) {
	key := cacheKey{region: region, number: s}

	cacheable := len(region)+len(s) <= maxPhoneCacheKeyLength
	if cacheable {
		if got, ok := phoneCache.get(key); ok {
			return got.value, !got.err
		}
	}

	n, err := phonenumbers.Parse(separators.Replace(s), region)
	if err != nil || !phonenumbers.IsValidNumber(n) {
		if cacheable {
			phoneCache.put(key, cacheEntry{err: true})
		}

		return "", false
	}

	out := phonenumbers.Format(n, phonenumbers.E164)
	if cacheable {
		phoneCache.put(key, cacheEntry{value: out})
	}

	return out, true
}
