// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import "strings"

// Parse parses a modifier clause such as trim or phone(region=US). Parameters
// are separated by semicolons; a parameter without '=' maps to an empty value.
// Every call returns a fresh parameter map, so callers may modify it.
func Parse(s string) Clause {
	s = strings.TrimSpace(s)

	i := strings.IndexByte(s, '(')
	if i < 0 {
		return Clause{Name: s}
	}

	name := strings.TrimSpace(s[:i])
	if name == "" || !strings.HasSuffix(s, ")") {
		return Clause{Name: s}
	}

	p := map[string]string{}

	for part := range strings.SplitSeq(s[i+1:len(s)-1], ";") {
		if part == "" {
			continue
		}

		k, v, _ := strings.Cut(part, "=")
		p[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	return Clause{Name: name, Params: p}
}
