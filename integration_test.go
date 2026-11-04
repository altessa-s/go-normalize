// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

//go:build integration

package normalize_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/altessa-s/go-normalize"
	"github.com/altessa-s/go-normalize/modifiers"
	"github.com/altessa-s/go-normalize/normalizetest"
)

type integrationAddress struct {
	Street  string  `normalize:"trim,remove_bad_symbols"`
	City    string  `normalize:"trim,uppercase"`
	Country *string `normalize:"trim,uppercase,nil_on_empty"`
}

type integrationContact struct {
	Kind  string `normalize:"trim,lowercase"`
	Value string `normalize:"trim"`
}

type integrationCustomer struct {
	ID        int
	Name      string   `normalize:"trim,remove_bad_symbols"`
	Email     *string  `normalize:"trim,lowercase,nil_on_empty"`
	Tags      []string `normalize:"remove_empty_elements"`
	Raw       string   `normalize:"-"`
	Display   string   `normalize:"custom"`
	Primary   *integrationAddress
	Addresses map[string]integrationAddress
	Contacts  []integrationContact
	Extra     any
	Manager   *integrationCustomer
}

// Normalize derives Display from the normalized fields, which are already
// processed when it runs.
func (c *integrationCustomer) Normalize() error {
	c.Display = c.Name
	if c.Primary != nil {
		c.Display += " (" + c.Primary.City + ")"
	}
	return nil
}

func newIntegrationCustomer(i int) *integrationCustomer {
	c := &integrationCustomer{
		ID:        i,
		Name:      "  Jane\u200b\u202e Doe " + strconv.Itoa(i) + "  ",
		Email:     new("  JANE@EXAMPLE.COM "),
		Tags:      []string{"vip", " ", "", "beta"},
		Raw:       "  untouched  ",
		Primary:   &integrationAddress{Street: " 1 Main St\u2028", City: " berlin ", Country: new(" ")},
		Addresses: map[string]integrationAddress{"home": {City: " paris ", Country: new(" fr ")}},
		Contacts:  []integrationContact{{Kind: " PHONE ", Value: " 123 "}},
		Extra:     integrationContact{Kind: " EMAIL ", Value: " x@y "},
	}
	c.Manager = c
	return c
}

func TestIntegrationNormalizesCustomerAggregate(t *testing.T) {
	t.Parallel()
	c := newIntegrationCustomer(1)
	normalizetest.RequireNormalized(t, normalize.Normalize, c)

	if c.Name != "Jane\u200b Doe 1" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Email == nil || *c.Email != "jane@example.com" {
		t.Errorf("Email = %v", c.Email)
	}
	if !reflect.DeepEqual(c.Tags, []string{"vip", "beta"}) {
		t.Errorf("Tags = %q", c.Tags)
	}
	if c.Raw != "  untouched  " {
		t.Errorf("Raw = %q", c.Raw)
	}
	if c.Primary.Street != "1 Main St" || c.Primary.City != "BERLIN" || c.Primary.Country != nil {
		t.Errorf("Primary = %+v", *c.Primary)
	}
	if home := c.Addresses["home"]; home.City != "PARIS" || home.Country == nil || *home.Country != "FR" {
		t.Errorf("Addresses[home] = %+v", home)
	}
	if c.Contacts[0] != (integrationContact{Kind: "phone", Value: "123"}) {
		t.Errorf("Contacts = %+v", c.Contacts)
	}
	if c.Extra != (integrationContact{Kind: "email", Value: "x@y"}) {
		t.Errorf("Extra = %+v", c.Extra)
	}
	if c.Display != "Jane\u200b Doe 1 (BERLIN)" {
		t.Errorf("Display = %q", c.Display)
	}
	if c.Manager != c {
		t.Error("self reference was replaced")
	}
}

func TestIntegrationNormalizeIsIdempotent(t *testing.T) {
	t.Parallel()
	first := newIntegrationCustomer(2)
	normalizetest.RequireNormalized(t, normalize.Normalize, first)
	snapshot := *first
	snapshot.Primary = new(*first.Primary)
	normalizetest.RequireNormalized(t, normalize.Normalize, first)
	if first.Name != snapshot.Name || *first.Primary != *snapshot.Primary || first.Display != snapshot.Display {
		t.Fatalf("second pass changed the value: %+v", first)
	}
}

func TestIntegrationConcurrentCallsShareRegistryAndPlans(t *testing.T) {
	t.Parallel()
	const workers, perWorker = 8, 50
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := range workers {
		wg.Go(func() {
			for i := range perWorker {
				c := newIntegrationCustomer(w*perWorker + i)
				if err := normalize.Normalize(c); err != nil {
					errs <- err
					return
				}
				if want := "Jane\u200b Doe " + strconv.Itoa(w*perWorker+i); c.Name != want {
					errs <- errors.New("got " + c.Name + ", want " + want)
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

func TestIntegrationCustomRegistryComposesBuiltins(t *testing.T) {
	t.Parallel()
	r := modifiers.NewRegistry()
	for _, name := range []string{"trim", "lowercase"} {
		m, ok := modifiers.DefaultRegistry().Get(name)
		if !ok {
			t.Fatalf("missing built-in %s", name)
		}
		r.Register(name, m)
	}
	r.Register("slug", func(v reflect.Value, params map[string]string) modifiers.ModifierResult {
		sep := params["sep"]
		return modifiers.ApplyStringModifier(v, func(s string) (string, bool) {
			out := strings.Join(strings.Fields(s), sep)
			return out, out != s
		})
	})
	type article struct {
		Title string `normalize:"trim,lowercase,slug(sep=-)"`
		Body  string `normalize:"uppercase"`
	}
	a := article{Title: "  Hello  Big World "}
	err := normalize.NormalizeWithRegistry(&a, r)
	if !errors.Is(err, normalize.ErrModifierNotFound) || !strings.Contains(err.Error(), "uppercase at Body") {
		t.Fatalf("got %v, want a missing uppercase modifier at Body", err)
	}
	if a.Title != "hello-big-world" {
		t.Fatalf("Title = %q", a.Title)
	}
}

func TestIntegrationErrorsCarryFieldContext(t *testing.T) {
	t.Parallel()
	r := modifiers.NewRegistry()
	r.Register("positive", func(v reflect.Value, _ map[string]string) modifiers.ModifierResult {
		if v.Int() <= 0 {
			return modifiers.ModifierResult{Error: &modifiers.ModifierError{Cause: strconv.ErrRange}}
		}
		return modifiers.ModifierResult{Value: v}
	})
	type line struct {
		Quantity int `normalize:"positive"`
	}
	type order struct{ Lines map[string][]line }
	err := normalize.NormalizeWithRegistry(&order{Lines: map[string][]line{"sku-secret": {{Quantity: 1}, {Quantity: 0}}}}, r)
	if !errors.Is(err, strconv.ErrRange) {
		t.Fatalf("cause lost: %v", err)
	}
	modifierErr, ok := errors.AsType[*modifiers.ModifierError](err)
	if !ok || modifierErr.FieldPath != "Lines.{key}[1].Quantity" || modifierErr.ModifierName != "positive" {
		t.Fatalf("unexpected context: %v", err)
	}
	if strings.Contains(err.Error(), "sku-secret") {
		t.Fatalf("map key leaked into the error: %v", err)
	}
}
