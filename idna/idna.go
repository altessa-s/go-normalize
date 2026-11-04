// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package idna

import (
	"errors"
	"net/url"
	"reflect"
	"strings"

	"github.com/altessa-s/go-normalize/modifiers"

	netidna "golang.org/x/net/idna"
)

// ErrInvalidDomain identifies a domain that is not a valid internationalized
// domain name. Errors do not include the value.
var ErrInvalidDomain = errors.New("domain is invalid")

// Registrar accepts modifier registrations. *modifiers.Registry implements it.
type Registrar interface {
	Register(name string, modifier modifiers.Modifier)
}

// Register adds the idna modifier to registry. Registration is explicit and
// has no package side effect. A nil registry is ignored.
func Register(registry Registrar) {
	if registry == nil {
		return
	}
	registry.Register("idna", ToASCII)
}

// ToASCII converts internationalized domain names to the ASCII form that DNS
// and mail servers use, by the UTS #46 lookup rules: the domain is mapped to
// lower case and non-ASCII labels become punycode ("xn--..."). It handles a
// bare domain, the domain after the last '@' of an email address and the
// host of an absolute URL; IP literals are left as they are. Empty values are
// left unchanged; an invalid domain fails with ErrInvalidDomain.
func ToASCII(v reflect.Value, _ map[string]string) modifiers.ModifierResult {
	failed := false
	out := modifiers.ApplyStringModifier(v, func(s string) (string, bool) {
		n, err := convert(s)
		if err != nil {
			failed = true
			return s, false
		}
		return n, n != s
	})
	if failed {
		return modifiers.ModifierResult{Value: v, Error: &modifiers.ModifierError{ModifierName: "idna", Cause: ErrInvalidDomain}}
	}
	return out
}

func convert(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.Host != "" && u.Opaque == "" {
		host := u.Hostname()
		if strings.Contains(host, ":") {
			return s, nil
		}
		ascii, err := netidna.Lookup.ToASCII(host)
		if err != nil {
			return s, err
		}
		if port := u.Port(); port != "" {
			ascii += ":" + port
		}
		u.Host = ascii
		return u.String(), nil
	}
	if local, domain, ok := strings.CutLast(s, "@"); ok {
		ascii, err := netidna.Lookup.ToASCII(domain)
		return local + "@" + ascii, err
	}
	return netidna.Lookup.ToASCII(s)
}
