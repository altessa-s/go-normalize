// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package idna_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/idna"
)

func BenchmarkToASCII(b *testing.B) {
	cases := []struct{ name, input string }{
		{name: "ascii", input: "user@example.com"},
		{name: "unicode", input: "user@\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444"},
		{name: "url", input: "https://example.com:8443/path"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			var s string
			v := reflect.ValueOf(&s).Elem()
			for b.Loop() {
				s = tc.input
				if out := idna.ToASCII(v, nil); out.Error != nil {
					b.Fatal(out.Error)
				}
			}
		})
	}
}
