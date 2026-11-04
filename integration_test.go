// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

//go:build integration

package normalize_test

import (
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
)

func TestIntegrationNestedDomain(t *testing.T) {
	type address struct {
		City string `normalize:"trim,uppercase"`
	}
	type person struct {
		Name    string `normalize:"trim"`
		Address *address
	}
	v := person{Name: " A ", Address: &address{City: " X "}}
	if err := normalize.Normalize(&v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "A" || v.Address.City != "X" {
		t.Fatalf("unexpected result: %#v", v)
	}
	_ = modifiers.DefaultRegistry()
}
