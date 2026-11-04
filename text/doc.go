// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package text provides opt-in Unicode modifiers for normalize tags: nfc,
// nfkc, fold and title(lang=xx). It is a separate module so that
// golang.org/x/text stays out of the root module's requirements.
//
//	registry := modifiers.NewRegistry()
//	text.Register(registry)
//
//	type Account struct {
//		Login string `normalize:"trim,nfkc,fold"`
//		Name  string `normalize:"trim,nfc,title(lang=en)"`
//	}
//
// # Forms
//
// nfc composes characters canonically and is safe on any text. nfkc also
// replaces compatibility characters and suits identifiers and search keys.
// fold maps case variants to one form for caseless keys.
package text
