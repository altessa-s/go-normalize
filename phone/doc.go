// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package phone provides an opt-in phone number modifier for normalize tags.
//
// Register adds the modifier, named phone, to a registry; nothing is
// registered implicitly. Valid numbers are formatted in E.164. Register takes
// the default region for numbers written without a country code; there is no
// built-in default, and the tag parameter region overrides it per field.
//
//	registry := modifiers.DefaultRegistry()
//	if err := phone.Register(registry, "US"); err != nil {
//		return err
//	}
//
//	type Contact struct {
//		Mobile string `normalize:"trim,phone"`
//		Office string `normalize:"trim,phone(region=DE)"`
//	}
//
// # Errors
//
// Numbers that do not parse or are invalid for the region fail with a
// *modifiers.Error that wraps ErrInvalidPhone. The error does not include
// the number. A region without phone number metadata fails with
// ErrUnsupportedRegion: from Register and New directly, and from a tag
// together with modifiers.ErrInvalidParameter.
package phone
