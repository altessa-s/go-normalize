// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/phone"
)

func FuzzNormalizePhone(f *testing.F) {
	f.Add("2025550123", "US")
	f.Add("", "RU")
	f.Fuzz(func(t *testing.T, s, region string) {
		v := reflect.ValueOf(s)
		result := phone.NormalizePhone(v, map[string]string{"region": region})
		if result.Value.IsValid() && result.Value.Kind() != reflect.String {
			t.Fatalf("unexpected kind %s", result.Value.Kind())
		}
	})
}
