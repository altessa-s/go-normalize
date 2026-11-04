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
var ErrInvalidDomain = errors.New("idna: invalid domain")

// Register adds the idna modifier to registry. Registration is explicit and
// has no package side effect. A nil registry, typed nil pointers included,
// fails with modifiers.ErrInvalidRegistration.
func Register(registry modifiers.Registrar) error {
	if err := modifiers.CheckRegistrar(registry); err != nil {
		return err
	}

	return registry.Register("idna", ToASCII)
}

// ToASCII converts internationalized domain names to the ASCII form that DNS
// and mail servers use, by the UTS #46 lookup rules: the domain is mapped to
// lower case and non-ASCII labels become punycode ("xn--..."). It handles a
// bare domain, the domain after the last '@' of an email address and the
// host of an absolute URL; IP literals are left as they are. Empty values are
// left unchanged; an invalid domain fails with ErrInvalidDomain.
func ToASCII(v reflect.Value, _ map[string]string) modifiers.Result {
	failed := false

	out := modifiers.ApplyString(v, func(s string) (string, bool) {
		n, err := convert(s)
		if err != nil {
			failed = true
			return s, false
		}

		return n, n != s
	})
	if failed {
		return modifiers.Result{Value: v, Err: ErrInvalidDomain}
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
