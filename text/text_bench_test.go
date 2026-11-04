// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package text_test

import (
	"reflect"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/text"
)

func BenchmarkModifiers(b *testing.B) {
	cases := []struct {
		name   string
		fn     modifiers.Modifier
		params map[string]string
		input  string
	}{
		{name: "nfc/normal", fn: text.NFC, input: "Normalization of plain text"},
		{name: "nfc/decomposed", fn: text.NFC, input: "Cafe\u0301 cre\u0300me"},
		{name: "nfkc", fn: text.NFKC, input: "\uff21\uff22\uff23 \ufb01le"},
		{name: "fold", fn: text.Fold, input: "Stra\u00dfe STRASSE"},
		{name: "title", fn: text.Title, params: map[string]string{"lang": "en"}, input: "hello world"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			var s string
			v := reflect.ValueOf(&s).Elem()
			for b.Loop() {
				s = tc.input
				if out := tc.fn(v, tc.params); out.Error != nil {
					b.Fatal(out.Error)
				}
			}
		})
	}
}
