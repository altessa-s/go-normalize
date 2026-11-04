// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package idna_test

import (
	"fmt"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/idna"
	"github.com/altessa-s/go-normalize/modifiers"
)

func ExampleRegister() {
	registry := modifiers.DefaultRegistry()
	if err := idna.Register(registry); err != nil {
		fmt.Println(err)
		return
	}

	type Contact struct {
		Email   string `normalize:"trim,email,idna"`
		Website string `normalize:"trim,url,idna"`
	}

	contact := Contact{Email: " user@B\u00fccher.de ", Website: " HTTPS://B\u00fccher.example:443/shop "}
	if err := normalize.NormalizeWithRegistry(&contact, registry); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(contact.Email, contact.Website)
	// Output: user@xn--bcher-kva.de https://xn--bcher-kva.example/shop
}
