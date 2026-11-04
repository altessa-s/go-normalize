// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import "testing"

func TestURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{input: " HTTPS://Example.COM ", want: "https://example.com/"},
		{input: "http://Example.com:80/Path?Q=V#Frag", want: "http://example.com/Path?Q=V#Frag"},
		{input: "https://example.com:443", want: "https://example.com/"},
		{input: "HTTPS://Example.com:443/", want: "https://example.com/"},
		{input: "HtTp://a:80", want: "http://a/"},
		{input: "WSS://Chat.Example.com:443/s", want: "wss://chat.example.com/s"},
		{input: "https://example.com:8443/a", want: "https://example.com:8443/a"},
		{input: "http://example.com:443/", want: "http://example.com:443/"},
		{input: "wss://Chat.Example.com:443/socket", want: "wss://chat.example.com/socket"},
		{input: "ftp://Files.Example.com", want: "ftp://files.example.com"},
		{input: "http://User:Pass@Example.com/", want: "http://User:Pass@example.com/"},
		{input: "http://[2001:DB8::1]:80/x", want: "http://[2001:db8::1]/x"},
		{input: "http://[2001:DB8::1]:8080", want: "http://[2001:db8::1]:8080/"},
		{input: "http://[FE80::1%25Eth0]:80/", want: "http://[fe80::1%25Eth0]/"},
		{input: "https://example.com/a%2Fb", want: "https://example.com/a%2Fb"},
		{input: "https://example.com/", want: "https://example.com/"},
		{input: "/relative/Path", want: "/relative/Path"},
		{input: " mailto:Someone@Example.COM ", want: "mailto:Someone@Example.COM"},
		{input: "http://exa mple.com", want: "http://exa mple.com"},
		{input: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got, err := applyNamed(t, "url", nil, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if got != tc.want {
				t.Fatalf("url(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
