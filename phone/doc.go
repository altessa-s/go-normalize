// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package phone provides an opt-in phone number modifier for normalize tags.
//
// Register adds the modifier, named phone, to a registry; nothing is
// registered implicitly. Valid numbers are formatted in E.164; the region
// parameter applies to numbers without a country code and defaults to RU.
//
//	registry := modifiers.NewRegistry()
//	phone.Register(registry)
//
//	type Contact struct {
//		Mobile string `normalize:"phone(region=US)"`
//	}
//
// # Errors
//
// Numbers that do not parse or are invalid for the region fail with a
// *modifiers.ModifierError whose cause is ErrInvalidPhone. The error does not
// include the number.
package phone
