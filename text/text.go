// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package text

import (
	"fmt"
	"reflect"

	"github.com/altessa-s/go-normalize/modifiers"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

// Register adds nfc, nfkc, fold and title to registry. Registration is
// explicit and has no package side effect. A nil registry fails with
// modifiers.ErrInvalidRegistration.
func Register(registry modifiers.Registrar) error {
	if registry == nil {
		return fmt.Errorf("%w: nil registry", modifiers.ErrInvalidRegistration)
	}

	for _, m := range []struct {
		name     string
		modifier modifiers.Modifier
	}{{"nfc", NFC}, {"nfkc", NFKC}, {"fold", Fold}, {"title", Title}} {
		if err := registry.Register(m.name, m.modifier); err != nil {
			return err
		}
	}

	return nil
}

// NFC applies Unicode canonical composition, so that text typed with a
// precomposed character and with a base letter plus a combining mark compares
// equal. Use it on any text that is compared, indexed or kept unique.
func NFC(v reflect.Value, _ map[string]string) modifiers.Result {
	return normalizeForm(v, norm.NFC)
}

// NFKC applies Unicode compatibility composition: besides NFC it replaces
// compatibility characters such as full-width letters, ligatures and
// superscripts with their plain forms. It changes how text looks, so use it
// for identifiers and search keys rather than prose.
func NFKC(v reflect.Value, _ map[string]string) modifiers.Result {
	return normalizeForm(v, norm.NFKC)
}

func normalizeForm(v reflect.Value, form norm.Form) modifiers.Result {
	return modifiers.ApplyString(v, func(s string) (string, bool) {
		if form.IsNormalString(s) {
			return s, false
		}

		return form.String(s), true
	})
}

// Fold applies Unicode case folding, which maps case variants to one form for
// caseless comparison: "Straße" and "STRASSE" both fold to "strasse".
// Unlike lowercase it is meant for keys, not for display.
func Fold(v reflect.Value, _ map[string]string) modifiers.Result {
	return applyCaser(v, cases.Fold())
}

// Title converts text to title case using the rules of the lang parameter, a
// BCP 47 tag such as en, nl or tr; without it the rules are language neutral.
// An invalid tag fails with modifiers.ErrInvalidParameter.
func Title(v reflect.Value, params map[string]string) modifiers.Result {
	tag := language.Und

	if raw, ok := params["lang"]; ok {
		parsed, err := language.Parse(raw)
		if err != nil {
			return modifiers.Result{
				Value: v,
				Err:   fmt.Errorf("%w: lang must be a BCP 47 tag, got %q", modifiers.ErrInvalidParameter, raw),
			}
		}

		tag = parsed
	}

	return applyCaser(v, cases.Title(tag))
}

// applyCaser runs c on string values. A Caser keeps state between calls, so
// each modifier call creates its own.
func applyCaser(v reflect.Value, c cases.Caser) modifiers.Result {
	return modifiers.ApplyString(v, func(s string) (string, bool) {
		n := c.String(s)
		return n, n != s
	})
}
