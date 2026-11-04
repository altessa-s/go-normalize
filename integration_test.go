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
	"time"

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

type integrationSignup struct {
	FullName string   `normalize:"trim,collapse_spaces,truncate(max=12)"`
	Email    string   `normalize:"email"`
	TaxID    *string  `normalize:"digits_only,nil_on_empty"`
	Phone    *string  `normalize:"digits_only,nil_on_zero"`
	Referrer *string  `normalize:"empty_on_nil,trim"`
	Age      *int     `normalize:"nil_on_zero"`
	Tags     []string `normalize:"remove_empty_elements,dedupe"`
	Scores   []int    `normalize:"dedupe"`
}

func TestIntegrationSignupModifiers(t *testing.T) {
	t.Parallel()

	s := integrationSignup{
		FullName: "  Anna \t Maria\u00a0 von   Ostrowski ",
		Email:    " Anna.M@Example.ORG ",
		TaxID:    new("no digits"),
		Phone:    new("+49 (30) 1234-567"),
		Age:      new(0),
		Tags:     []string{"beta", " ", "vip", "beta"},
		Scores:   []int{10, 7, 10},
	}
	normalizetest.RequireNormalized(t, normalize.Normalize, &s)

	if s.FullName != "Anna Maria v" {
		t.Errorf("FullName = %q", s.FullName)
	}

	if s.Email != "Anna.M@example.org" {
		t.Errorf("Email = %q", s.Email)
	}

	if s.TaxID != nil || s.Phone == nil || *s.Phone != "49301234567" {
		t.Errorf("TaxID = %v, Phone = %v", s.TaxID, s.Phone)
	}

	if s.Referrer == nil || *s.Referrer != "" || s.Age != nil {
		t.Errorf("Referrer = %v, Age = %v", s.Referrer, s.Age)
	}

	if !reflect.DeepEqual(s.Tags, []string{"beta", "vip"}) || !reflect.DeepEqual(s.Scores, []int{10, 7}) {
		t.Errorf("Tags = %v, Scores = %v", s.Tags, s.Scores)
	}
}

func TestIntegrationInvalidModifierParameter(t *testing.T) {
	t.Parallel()

	type post struct {
		Title string `normalize:"truncate(max=lots)"`
	}

	err := normalize.Normalize(&post{Title: "title"})
	if !errors.Is(err, modifiers.ErrInvalidParameter) {
		t.Fatalf("got %v, want ErrInvalidParameter", err)
	}

	if modifierErr, ok := errors.AsType[*modifiers.ModifierError](err); !ok || modifierErr.FieldPath != "Title" || modifierErr.ModifierName != "truncate" {
		t.Fatalf("unexpected context: %v", err)
	}
}

func TestIntegrationApplyModifierOnNumbers(t *testing.T) {
	t.Parallel()

	type percent int

	r := modifiers.NewRegistry()
	r.Register("percent", func(v reflect.Value, _ map[string]string) modifiers.ModifierResult {
		return modifiers.ApplyModifier(v, func(n int) (int, bool) {
			c := min(max(n, 0), 100)
			return c, c != n
		})
	})

	type report struct {
		Done   percent  `normalize:"percent"`
		Failed *percent `normalize:"percent"`
		Rows   []struct {
			Share int `normalize:"percent"`
		}
	}

	v := report{Done: 140, Failed: new(percent(-5))}

	v.Rows = append(v.Rows, struct {
		Share int `normalize:"percent"`
	}{Share: 101})
	if err := normalize.NormalizeWithRegistry(&v, r); err != nil {
		t.Fatal(err)
	}

	if v.Done != 100 || *v.Failed != 0 || v.Rows[0].Share != 100 {
		t.Fatalf("unexpected result: %+v, Failed=%d", v, *v.Failed)
	}
}

type integrationWebhook struct {
	Endpoint   string     `normalize:"trim,url"`
	Fallback   *string    `normalize:"url,nil_on_empty"`
	CreatedAt  time.Time  `normalize:"utc,truncate_time(unit=us)"`
	ExpiresAt  *time.Time `normalize:"utc,truncate_time(unit=1s)"`
	Retries    int        `normalize:"clamp(min=0;max=10)"`
	Timeout    float64    `normalize:"clamp(min=0.5;max=30)"`
	Priority   *uint8     `normalize:"clamp(max=9)"`
	Deliveries []struct {
		At     time.Time `normalize:"utc"`
		Status int       `normalize:"clamp(min=100;max=599)"`
	}
}

func TestIntegrationWebhookModifiers(t *testing.T) {
	t.Parallel()

	zone := time.FixedZone("UTC+3", 3*3600)
	created := time.Date(2026, 5, 6, 10, 0, 0, 123456789, zone)
	expires := time.Date(2026, 6, 6, 10, 0, 0, 999999999, zone)
	w := integrationWebhook{
		Endpoint:  "  HTTPS://Hooks.Example.COM:443/in?Token=AbC#Top ",
		Fallback:  new(""),
		CreatedAt: created,
		ExpiresAt: new(expires),
		Retries:   99,
		Timeout:   0.1,
		Priority:  new(uint8(200)),
	}
	w.Deliveries = append(w.Deliveries, struct {
		At     time.Time `normalize:"utc"`
		Status int       `normalize:"clamp(min=100;max=599)"`
	}{At: created, Status: 999})
	normalizetest.RequireNormalized(t, normalize.Normalize, &w)

	if w.Endpoint != "https://hooks.example.com/in?Token=AbC#Top" || w.Fallback != nil {
		t.Errorf("Endpoint = %q, Fallback = %v", w.Endpoint, w.Fallback)
	}

	if w.CreatedAt.Location() != time.UTC || w.CreatedAt.Nanosecond() != 123456000 || !w.CreatedAt.Equal(created.Truncate(time.Microsecond)) {
		t.Errorf("CreatedAt = %v", w.CreatedAt)
	}

	if w.ExpiresAt.Location() != time.UTC || w.ExpiresAt.Nanosecond() != 0 {
		t.Errorf("ExpiresAt = %v", *w.ExpiresAt)
	}

	if w.Retries != 10 || w.Timeout != 0.5 || *w.Priority != 9 {
		t.Errorf("Retries = %d, Timeout = %v, Priority = %d", w.Retries, w.Timeout, *w.Priority)
	}

	if d := w.Deliveries[0]; d.At.Location() != time.UTC || d.Status != 599 {
		t.Errorf("Deliveries[0] = %+v", d)
	}
}

func TestIntegrationClampBoundOutsideFieldType(t *testing.T) {
	t.Parallel()

	type config struct {
		Workers []struct {
			Weight int8 `normalize:"clamp(max=1000)"`
		}
	}

	c := config{Workers: []struct {
		Weight int8 `normalize:"clamp(max=1000)"`
	}{{Weight: 5}}}

	err := normalize.Normalize(&c)
	if !errors.Is(err, modifiers.ErrInvalidParameter) {
		t.Fatalf("got %v, want ErrInvalidParameter", err)
	}

	if modifierErr, ok := errors.AsType[*modifiers.ModifierError](err); !ok || modifierErr.FieldPath != "Workers[0].Weight" {
		t.Fatalf("unexpected context: %v", err)
	}
}
