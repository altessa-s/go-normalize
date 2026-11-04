// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize_test

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
)

func ExampleNormalize() {
	type Contact struct {
		Name  string   `normalize:"trim,collapse_spaces"`
		Email *string  `normalize:"trim,email,nil_on_empty"`
		Tags  []string `normalize:"remove_empty_elements,dedupe"`
	}

	email := "  Jane.Doe@Example.COM "
	contact := Contact{Name: "  Jane   Doe ", Email: &email, Tags: []string{"vip", " ", "vip", "new"}}

	if err := normalize.Normalize(&contact); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%q %q %q\n", contact.Name, *contact.Email, contact.Tags)
	// Output: "Jane Doe" "Jane.Doe@example.com" ["vip" "new"]
}

func ExampleNormalizeWithRegistry() {
	registry := modifiers.DefaultRegistry()

	err := registry.Register("slug", func(v reflect.Value, params map[string]string) modifiers.Result {
		return modifiers.ApplyString(v, func(s string) (string, bool) {
			out := strings.Join(strings.Fields(s), params["sep"])
			return out, out != s
		})
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	type Article struct {
		Title string `normalize:"lowercase,slug(sep=-)"`
	}

	article := Article{Title: "  Hello  Normalize World "}
	if err := normalize.NormalizeWithRegistry(&article, registry); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(article.Title)
	// Output: hello-normalize-world
}

// Period is a CustomNormalizer: its Normalize method runs after its tagged
// fields.
type Period struct {
	Label    string `normalize:"trim"`
	From, To int
}

// Normalize orders the bounds.
func (p *Period) Normalize() error {
	if p.From > p.To {
		p.From, p.To = p.To, p.From
	}

	return nil
}

func ExampleCustomNormalizer() {
	type Report struct {
		Periods []Period
	}

	report := Report{Periods: []Period{{Label: " Q1 ", From: 3, To: 1}}}
	if err := normalize.Normalize(&report); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v\n", report.Periods[0])
	// Output: {Label:Q1 From:1 To:3}
}
