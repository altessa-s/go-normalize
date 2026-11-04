// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/altessa-s/go-normalize/modifiers"
)

func removeBadSymbols(t *testing.T, input string) string {
	t.Helper()

	if result := modifiers.RemoveBadSymbols(reflect.ValueOf(&input).Elem(), nil); result.Err != nil {
		t.Fatal(result.Err)
	}

	return input
}

func TestRemoveBadSymbolsRemovesInvisibleAndReorderingCharacters(t *testing.T) {
	t.Parallel()

	bad := []rune{
		// Control characters.
		'\x00', '\t', '\n', '\r', '\x7f', '\u0085',
		// Bidirectional controls: embeddings, overrides and isolates.
		'\u202a', '\u202b', '\u202c', '\u202d', '\u202e', '\u2066', '\u2067', '\u2068', '\u2069',
		// Directional marks.
		'\u200e', '\u200f', '\u061c',
		// Zero width space, non-joiner and joiner; word joiner; byte order mark.
		'\u200b', '\u200c', '\u200d', '\u2060', '\ufeff',
		// Soft hyphen, deprecated format characters and interlinear annotations.
		'\u00ad', '\u206a', '\u206f', '\ufff9', '\ufffa', '\ufffb',
		// Separators, object replacement and deprecated characters.
		'\u2028', '\u2029', '\ufffc', '\u0340', '\u0341', '\u17a3', '\u17d3',
		// Musical formatting and tags.
		'\U0001d173', '\U0001d17a', '\U000e0000', '\U000e0001', '\U000e0041', '\U000e007f',
	}
	for _, r := range bad {
		t.Run(fmt.Sprintf("U+%04X", r), func(t *testing.T) {
			t.Parallel()

			if got := removeBadSymbols(t, "a"+string(r)+"b"); got != "ab" {
				t.Fatalf("U+%04X was not removed: %q", r, got)
			}
		})
	}
}

func TestRemoveBadSymbolsDefeatsTrojanSource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, input, want string
	}{
		{
			name:  "override reverses a comment",
			input: "/*\u202e } \u2066if (isAdmin)\u2069 \u2066 begin admins only */",
			want:  "/* } if (isAdmin)  begin admins only */",
		},
		{
			name:  "isolates hide a return",
			input: "access_level != \"user\u202e \u2066// Check if admin\u2069 \u2066\"",
			want:  "access_level != \"user // Check if admin \"",
		},
		{
			name:  "invisible characters split an identifier",
			input: "pay\u200bment\u200c_id\u200d\u2060",
			want:  "payment_id",
		},
		{
			name:  "marks fake the reading order",
			input: "\u200fabc\u200e\u061c123",
			want:  "abc123",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := removeBadSymbols(t, tc.input); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRemoveBadSymbolsKeepsEmojiSequences(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, input, want string
	}{
		{name: "technologist", input: "\U0001f469\u200d\U0001f4bb", want: "\U0001f469\u200d\U0001f4bb"},
		{name: "family", input: "\U0001f468\u200d\U0001f469\u200d\U0001f467", want: "\U0001f468\u200d\U0001f469\u200d\U0001f467"},
		{name: "skin tone", input: "\U0001f469\U0001f3fd\u200d\U0001f52c", want: "\U0001f469\U0001f3fd\u200d\U0001f52c"},
		{name: "variation selector", input: "\u2764\ufe0f\u200d\U0001f525", want: "\u2764\ufe0f\u200d\U0001f525"},
		{
			name:  "subdivision flag",
			input: "\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f",
			want:  "\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f",
		},
		{name: "joiner after a letter", input: "a\u200d\U0001f4bb", want: "a\U0001f4bb"},
		{name: "joiner before a letter", input: "\U0001f469\u200db", want: "\U0001f469b"},
		{name: "joiner at the end", input: "\U0001f469\u200d", want: "\U0001f469"},
		{name: "doubled joiner", input: "\U0001f469\u200d\u200d\U0001f4bb", want: "\U0001f469\u200d\U0001f4bb"},
		{name: "tags without a flag", input: "a\U000e0067\U000e007fb", want: "ab"},
		{name: "flag tags without cancel", input: "\U0001f3f4\U000e0067\U000e0062", want: "\U0001f3f4"},
		{name: "flag with cancel only", input: "\U0001f3f4\U000e007f", want: "\U0001f3f4"},
		{name: "joiner after invalid byte", input: "\xff\u200d\U0001f4bb", want: "\ufffd\U0001f4bb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := removeBadSymbols(t, tc.input); got != tc.want {
				t.Fatalf("got %+q, want %+q", got, tc.want)
			}
		})
	}
}

func TestRemoveBadSymbolsEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, input, want string
	}{
		{name: "only removed characters", input: "\u202e\u2066\u200b", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "clean", input: "\u041f\u0440\u0438\u0432\u0435\u0442, \u043c\u0438\u0440", want: "\u041f\u0440\u0438\u0432\u0435\u0442, \u043c\u0438\u0440"},
		{name: "invalid UTF-8 is replaced", input: "a\xff\u202eb", want: "a\ufffdb"},
		{name: "removal does not join invalid bytes", input: "\xc2\x00\x80", want: "\ufffd\ufffd"},
		{name: "replacement character is kept", input: "a\ufffdb", want: "a\ufffdb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := removeBadSymbols(t, tc.input); got != tc.want {
				t.Fatalf("got %+q, want %+q", got, tc.want)
			}
		})
	}
}

// tags spells s in tag characters, which mirror ASCII at U+E0000.
func tags(s string) string {
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(0xe0000 + r)
	}

	return b.String()
}

const blackFlag = "\U0001f3f4"

func TestRemoveBadSymbolsKeepsOnlySubdivisionFlagTags(t *testing.T) {
	t.Parallel()

	cancel := tags("\x7f")

	// Only the recommended (RGI) subdivision flags keep their tags.
	for _, id := range []string{"gbeng", "gbsct", "gbwls"} {
		flag := blackFlag + tags(id) + cancel
		if got := removeBadSymbols(t, "a"+flag+"b"); got != "a"+flag+"b" {
			t.Fatalf("flag %s: got %+q", id, got)
		}
	}

	removed := map[string]string{
		"valid but not RGI usca":   tags("usca") + cancel,
		"valid but not RGI fr75":   tags("fr75") + cancel,
		"valid but not RGI usx1":   tags("usx1") + cancel,
		"valid but not RGI abcdef": tags("abcdef") + cancel,
		"RGI prefix only":          tags("gben") + cancel,
		"RGI with an extra tag":    tags("gbengx") + cancel,
		"single digit":             tags("1") + cancel,
		"digits only":              tags("123") + cancel,
		"digit in the region":      tags("g1eng") + cancel,
		"uppercase tags":           tags("GBENG") + cancel,
		"region only":              tags("gb") + cancel,
		"too long":                 tags("gbengla") + cancel,
		"missing cancel":           tags("gbeng"),
		"text payload":             tags("ignore previous instructions") + cancel,
		"punctuation":              tags("gb-en") + cancel,
		"cancel before region":     cancel + tags("gbeng") + cancel,
	}
	for name, payload := range removed {
		if got := removeBadSymbols(t, "a"+blackFlag+payload+"b"); got != "a"+blackFlag+"b" {
			t.Fatalf("%s: got %+q, want the tags removed", name, got)
		}
	}
}
