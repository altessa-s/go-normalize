// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

//go:build integration

package phone_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/phone"
)

// newPhoneRegistry composes the built-ins a consumer needs with the phone modifier.
func newPhoneRegistry(t testing.TB) *modifiers.Registry {
	t.Helper()

	r := modifiers.NewRegistry()

	for _, name := range []string{"trim", "nil_on_empty"} {
		m, ok := modifiers.DefaultRegistry().Get(name)
		if !ok {
			t.Fatalf("missing built-in %s", name)
		}

		r.Register(name, m)
	}

	phone.Register(r)

	return r
}

type integrationContact struct {
	Name   string    `normalize:"trim"`
	Mobile string    `normalize:"trim,phone(region=US)"`
	Work   *string   `normalize:"trim,nil_on_empty,phone(region=DE)"`
	Home   *string   `normalize:"trim,nil_on_empty,phone"`
	Others []*string `normalize:"-"`
}

type integrationBook struct {
	Owner    integrationContact
	Contacts []integrationContact
	ByTeam   map[string]*integrationContact
}

func TestIntegrationPhoneWithBuiltins(t *testing.T) {
	t.Parallel()
	r := newPhoneRegistry(t)

	book := integrationBook{
		Owner: integrationContact{Name: " Owner ", Mobile: " (202) 555-0123 ", Work: new(" 030 1234567 "), Home: new("  ")},
		Contacts: []integrationContact{
			{Name: " A ", Mobile: "202.555.0124", Home: new("8 (916) 123-45-67")},
		},
		ByTeam: map[string]*integrationContact{"ops": {Mobile: "+1 202 555 0125"}},
	}
	if err := normalize.NormalizeWithRegistry(&book, r); err != nil {
		t.Fatal(err)
	}

	if book.Owner.Name != "Owner" || book.Owner.Mobile != "+12025550123" || *book.Owner.Work != "+49301234567" || book.Owner.Home != nil {
		t.Fatalf("Owner = %+v", book.Owner)
	}

	if c := book.Contacts[0]; c.Mobile != "+12025550124" || *c.Home != "+79161234567" {
		t.Fatalf("Contacts[0] = %+v", c)
	}

	if got := book.ByTeam["ops"].Mobile; got != "+12025550125" {
		t.Fatalf("ByTeam[ops].Mobile = %q", got)
	}
}

func TestIntegrationPhoneErrorContext(t *testing.T) {
	t.Parallel()
	r := newPhoneRegistry(t)
	book := integrationBook{Contacts: []integrationContact{{Mobile: "202-555-0123"}, {Mobile: "555-0100"}}}

	err := normalize.NormalizeWithRegistry(&book, r)
	if !errors.Is(err, phone.ErrInvalidPhone) {
		t.Fatalf("got %v, want ErrInvalidPhone", err)
	}

	modifierErr, ok := errors.AsType[*modifiers.ModifierError](err)
	if !ok || modifierErr.FieldPath != "Contacts[1].Mobile" || modifierErr.ModifierName != "phone" {
		t.Fatalf("unexpected context: %v", err)
	}

	if book.Contacts[0].Mobile != "+12025550123" {
		t.Fatalf("fields before the failure were not normalized: %+v", book.Contacts[0])
	}
}

// TestIntegrationPhoneConcurrentCacheChurn drives more distinct numbers than
// the cache holds from several goroutines, so lookups, inserts and evictions
// interleave under the race detector.
func TestIntegrationPhoneConcurrentCacheChurn(t *testing.T) {
	t.Parallel()
	r := newPhoneRegistry(t)

	const workers, perWorker = 8, 1200

	var wg sync.WaitGroup

	errs := make(chan error, workers)
	for w := range workers {
		wg.Go(func() {
			for i := range perWorker {
				n := (w*perWorker + i) % 10000

				c := integrationContact{Mobile: fmt.Sprintf("202-555-%04d", n)}
				if err := normalize.NormalizeWithRegistry(&c, r); err != nil {
					errs <- err
					return
				}

				if want := fmt.Sprintf("+1202555%04d", n); c.Mobile != want {
					errs <- fmt.Errorf("got %s, want %s", c.Mobile, want)
					return
				}
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}
