// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalizetest_test

import (
	"errors"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/normalizetest"
)

// fatalRecorder records Fatal instead of stopping the test.
type fatalRecorder struct {
	testing.TB
	fatal []any
}

func (r *fatalRecorder) Helper() {}

func (r *fatalRecorder) Fatal(args ...any) { r.fatal = args }

func TestRequireNormalizedPassesValueThrough(t *testing.T) {
	t.Parallel()

	value := struct {
		Name string `normalize:"trim"`
	}{Name: " x "}
	normalizetest.RequireNormalized(t, normalize.Normalize, &value)

	if value.Name != "x" {
		t.Fatalf("got %q", value.Name)
	}
}

func TestRequireNormalizedFailsOnError(t *testing.T) {
	t.Parallel()

	want := errors.New("rejected")
	r := &fatalRecorder{TB: t}
	normalizetest.RequireNormalized(r, func(any) error { return want }, nil)

	if len(r.fatal) != 1 || r.fatal[0] != want {
		t.Fatalf("Fatal called with %v, want %v", r.fatal, want)
	}
}
