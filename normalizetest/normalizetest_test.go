// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalizetest

import "testing"

func TestRequireNormalized(t *testing.T) { RequireNormalized(t, func(any) error { return nil }, nil) }

func TestRequireNormalizedAllowsNilValue(t *testing.T) {
	t.Parallel()
	RequireNormalized(t, func(any) error { return nil }, struct{}{})
}
