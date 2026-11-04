// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package text_test

import (
	"fmt"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/text"
)

func ExampleRegister() {
	registry := modifiers.DefaultRegistry()
	if err := text.Register(registry); err != nil {
		fmt.Println(err)
		return
	}

	type Account struct {
		Login string `normalize:"trim,nfkc,fold"`
		Name  string `normalize:"trim,nfc,title(lang=en)"`
	}

	account := Account{Login: " STRASSE ", Name: " jane doe "}
	if err := normalize.NormalizeWithRegistry(&account, registry); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(account.Login, account.Name)
	// Output: strasse Jane Doe
}
