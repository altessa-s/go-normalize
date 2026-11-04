// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package phone_test

import (
	"fmt"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/phone"
)

func ExampleRegister() {
	registry := modifiers.DefaultRegistry()
	if err := phone.Register(registry, "US"); err != nil {
		fmt.Println(err)
		return
	}

	type Contact struct {
		Mobile string `normalize:"trim,phone"`
		Office string `normalize:"trim,phone(region=DE)"`
	}

	contact := Contact{Mobile: " (202) 555-0123 ", Office: "030 1234567"}
	if err := normalize.NormalizeWithRegistry(&contact, registry); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(contact.Mobile, contact.Office)
	// Output: +12025550123 +49301234567
}
