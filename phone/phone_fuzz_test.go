// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/phone"
)

func FuzzPhone(f *testing.F) {
	m, err := phone.New("RU")
	if err != nil {
		f.Fatal(err)
	}

	f.Add("2025550123", "US")
	f.Add("", "RU")
	f.Add("+7 916 123-45-67", "")
	f.Fuzz(func(t *testing.T, s, region string) {
		v := reflect.ValueOf(s)

		result := m(v, map[string]string{"region": region})
		if result.Value.Kind() != reflect.String {
			t.Fatalf("unexpected kind %s", result.Value.Kind())
		}

		if result.Err != nil {
			if result.Value.String() != s {
				t.Fatalf("a failure changed the value: %v", result.Err)
			}

			return
		}

		if out := result.Value.String(); out != "" && out[0] != '+' {
			t.Fatalf("result %q is not in E.164", out)
		}
	})
}
