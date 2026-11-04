// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

//go:build integration

package phone_test

import "testing"

func TestPhoneIntegrationSeed(t *testing.T) { t.Parallel(); TestRegisterAndNormalize(t) }
