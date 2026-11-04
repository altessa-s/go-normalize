// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/altessa-s/go-normalize/modifiers"
)

func TestUTC(t *testing.T) {
	t.Parallel()

	local := time.Date(2026, 3, 4, 5, 6, 7, 8, time.FixedZone("UTC+3", 3*3600))
	v := struct {
		At      time.Time
		Pointer *time.Time
		Nil     *time.Time
		Zero    time.Time
	}{At: local, Pointer: new(local)}

	utc := must(modifiers.DefaultRegistry().Get("utc"))
	for i := range reflect.TypeOf(v).NumField() {
		if out := utc(reflect.ValueOf(&v).Elem().Field(i), nil); out.Err != nil {
			t.Fatal(out.Err)
		}
	}

	if v.At.Location() != time.UTC || !v.At.Equal(local) || v.Pointer.Location() != time.UTC || !v.Pointer.Equal(local) {
		t.Fatalf("not converted: %v, %v", v.At, *v.Pointer)
	}

	if v.Nil != nil || !v.Zero.IsZero() {
		t.Fatalf("nil or zero value changed: %v, %v", v.Nil, v.Zero)
	}
}

func TestTruncateTime(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 3, 4, 5, 6, 7, 891234567, time.UTC)
	cases := []struct {
		unit string
		want time.Time
	}{
		{unit: "us", want: time.Date(2026, 3, 4, 5, 6, 7, 891234000, time.UTC)},
		{unit: "\u00b5s", want: time.Date(2026, 3, 4, 5, 6, 7, 891234000, time.UTC)},
		{unit: "ms", want: time.Date(2026, 3, 4, 5, 6, 7, 891000000, time.UTC)},
		{unit: "1s", want: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)},
		{unit: "15m", want: time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)},
		{unit: "ns", want: at},
		{unit: ".5s", want: time.Date(2026, 3, 4, 5, 6, 7, 500000000, time.UTC)},
		{unit: "+1s", want: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)},
	}
	truncate := must(modifiers.DefaultRegistry().Get("truncate_time"))

	for _, tc := range cases {
		t.Run(tc.unit, func(t *testing.T) {
			t.Parallel()

			got := at
			if out := truncate(reflect.ValueOf(&got).Elem(), map[string]string{"unit": tc.unit}); out.Err != nil {
				t.Fatal(out.Err)
			}

			if !got.Equal(tc.want) {
				t.Fatalf("truncate_time(unit=%s) = %v, want %v", tc.unit, got, tc.want)
			}
		})
	}

	for _, unit := range []string{"", "0s", "-1s", "-s", "+", ".", "fortnight"} {
		got := at

		out := truncate(reflect.ValueOf(&got).Elem(), map[string]string{"unit": unit})
		if !errors.Is(out.Err, modifiers.ErrInvalidParameter) || !got.Equal(at) {
			t.Fatalf("unit %q: got %v, value %v", unit, out.Err, got)
		}
	}
}
