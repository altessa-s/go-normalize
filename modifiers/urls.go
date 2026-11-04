// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"net/url"
	"reflect"
	"strings"
)

// defaultPorts are the ports that URL drops for their scheme.
var defaultPorts = map[string]string{"http": "80", "https": "443", "ws": "80", "wss": "443"}

// URL trims a URL and, when it is absolute with a host, applies the syntax
// normalizations of RFC 3986 section 6.2.2-6.2.3: it lower-cases the scheme
// and host, drops the default port of http, https, ws and wss, and gives an
// empty http or https path the root path "/". Query and fragment are kept.
// Relative references, opaque URLs such as mailto: and values that do not
// parse are only trimmed. It does not validate the URL.
func URL(v reflect.Value, _ map[string]string) Result {
	return ApplyString(v, normalizeURL)
}

func normalizeURL(s string) (string, bool) {
	n := strings.TrimSpace(s)

	u, err := url.Parse(n)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Opaque != "" {
		return n, n != s
	}
	// Lower-case the scheme here rather than rely on url.Parse doing so; the
	// default-port and root-path rules below depend on it.
	u.Scheme = strings.ToLower(u.Scheme)

	host, port := lowerHost(u.Hostname()), u.Port()
	if port == defaultPorts[u.Scheme] {
		port = ""
	}

	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	if port != "" {
		host += ":" + port
	}

	u.Host = host
	if u.Path == "" && (u.Scheme == "http" || u.Scheme == "https") {
		u.Path = "/"
	}

	out := u.String()

	return out, out != s
}

// lowerHost lower-cases a host name or IP literal. The zone of an IPv6 literal,
// after '%', names a network interface and keeps its case.
func lowerHost(host string) string {
	addr, zone, ok := strings.Cut(host, "%")
	if !ok {
		return strings.ToLower(host)
	}

	return strings.ToLower(addr) + "%" + zone
}
