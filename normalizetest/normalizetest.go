// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalizetest

import "testing"

// RequireNormalized runs Normalize and fails the test on error.
func RequireNormalized(t testing.TB, normalize func(any) error, value any) {
	t.Helper()
	if err := normalize(value); err != nil {
		t.Fatal(err)
	}
}
