// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"reflect"
	"time"
	"unicode"
	"unicode/utf8"
)

// UTC converts time.Time values to UTC; the instant is unchanged.
func UTC(v reflect.Value, _ map[string]string) Result {
	return Apply(v, func(t time.Time) (time.Time, bool) {
		return t.UTC(), t.Location() != time.UTC
	})
}

// TruncateTime rounds time.Time values down to a multiple of the unit
// parameter, a duration such as 1s, 500ms, .5s or 1h; a bare unit such as s,
// ms or us means one of it. Rounding counts from the zero time, as
// time.Time.Truncate does. Use it to match the precision of a store, such as
// microseconds for PostgreSQL. A missing or non-positive unit fails with
// ErrInvalidParameter.
func TruncateTime(v reflect.Value, params map[string]string) Result {
	raw := params["unit"]

	unit, err := parseUnit(raw)
	if err != nil || unit <= 0 {
		return invalidParameter(v, "unit must be a positive duration such as 1s or ms, got %q", raw)
	}

	return Apply(v, func(t time.Time) (time.Time, bool) {
		r := t.Truncate(unit)
		return r, !r.Equal(t)
	})
}

// parseUnit parses a duration; a bare unit such as ms, one starting with a
// letter, means one of it.
func parseUnit(s string) (time.Duration, error) {
	if r, _ := utf8.DecodeRuneInString(s); unicode.IsLetter(r) {
		s = "1" + s
	}

	return time.ParseDuration(s)
}
