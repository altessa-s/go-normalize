// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package idna provides an opt-in modifier that converts internationalized
// domain names to their ASCII form for normalize tags. It is a separate module
// so that golang.org/x/net stays out of the root module's requirements.
//
//	registry := modifiers.DefaultRegistry()
//	if err := idna.Register(registry); err != nil {
//		return err
//	}
//
//	type Contact struct {
//		Email   string `normalize:"trim,email,idna"`
//		Website string `normalize:"trim,url,idna"`
//	}
//
// # Errors
//
// A domain that is not a valid internationalized domain name fails with a
// *modifiers.Error whose cause is ErrInvalidDomain. The error does
// not include the value.
package idna
