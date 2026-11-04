// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"cmp"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// Clamp limits integers and floating-point numbers, named types and pointers
// included, to the range given by the min and max parameters. Either bound may
// be omitted, but not both. Bounds are parsed for the field's own type, so a
// bound outside that type's range, a non-numeric bound or min above max fails
// with ErrInvalidParameter. NaN is left unchanged.
func Clamp(v reflect.Value, params map[string]string) ModifierResult {
	lo, hasLo := params["min"]

	hi, hasHi := params["max"]
	if !hasLo && !hasHi {
		return invalidParameter(v, "clamp needs min, max or both")
	}

	var boundsErr error

	out := applyToLeaf(v, isNumber, func(src, dst reflect.Value) bool {
		changed, err := clampValue(src, dst, limits{lo: lo, hi: hi, hasLo: hasLo, hasHi: hasHi})
		boundsErr = err

		return changed
	})
	if boundsErr != nil {
		return invalidParameter(v, "%v", boundsErr)
	}

	return out
}

// limits holds the raw clamp bounds from the tag.
type limits struct {
	lo, hi       string
	hasLo, hasHi bool
}

func isNumber(t reflect.Type) bool {
	switch t.Kind() { //nolint:exhaustive // only numeric kinds are clamped
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	}

	return false
}

func clampValue(src, dst reflect.Value, l limits) (bool, error) {
	bits := src.Type().Bits()
	switch src.Kind() { //nolint:exhaustive // isNumber admits only these kinds
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		lo, hi, err := parseBounds(l, math.MinInt64, math.MaxInt64, func(s string) (int64, error) { return strconv.ParseInt(s, 10, bits) })
		if err != nil {
			return false, err
		}

		x := src.Int()
		if c := min(max(x, lo), hi); c != x {
			dst.SetInt(c)
			return true, nil
		}
	case reflect.Float32, reflect.Float64:
		lo, hi, err := parseBounds(l, math.Inf(-1), math.Inf(1), func(s string) (float64, error) { return parseFloatBound(s, bits) })
		if err != nil {
			return false, err
		}

		x := src.Float()
		if c := min(max(x, lo), hi); c != x && !math.IsNaN(x) {
			dst.SetFloat(c)
			return true, nil
		}
	default:
		lo, hi, err := parseBounds(l, 0, math.MaxUint64, func(s string) (uint64, error) { return strconv.ParseUint(s, 10, bits) })
		if err != nil {
			return false, err
		}

		x := src.Uint()
		if c := min(max(x, lo), hi); c != x {
			dst.SetUint(c)
			return true, nil
		}
	}

	return false, nil
}

func parseFloatBound(s string, bits int) (float64, error) {
	f, err := strconv.ParseFloat(s, bits)
	if err == nil && math.IsNaN(f) {
		return 0, fmt.Errorf("%q is not a number", s)
	}

	return f, err
}

// parseBounds parses the bounds present in l; an absent bound is the type's extreme.
func parseBounds[N cmp.Ordered](l limits, lowest, highest N, parse func(string) (N, error)) (N, N, error) {
	lo, hi := lowest, highest

	var err error
	if l.hasLo {
		if lo, err = parse(l.lo); err != nil {
			return lo, hi, fmt.Errorf("min: %w", err)
		}
	}

	if l.hasHi {
		if hi, err = parse(l.hi); err != nil {
			return lo, hi, fmt.Errorf("max: %w", err)
		}
	}

	if lo > hi {
		return lo, hi, fmt.Errorf("min %v exceeds max %v", lo, hi)
	}

	return lo, hi, nil
}
