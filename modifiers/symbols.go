// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package modifiers

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	zeroWidthJoiner     = '\u200d'
	variationSelector16 = '\ufe0f'
	blackFlag           = '\U0001f3f4'
)

// rgiSubdivisionFlags are the tag sequences that follow U+1F3F4 in the
// recommended (RGI) emoji subdivision flags of Unicode's emoji-sequences.txt:
// England (gbeng), Scotland (gbsct) and Wales (gbwls), each ending in the
// cancel tag U+E007F.
var rgiSubdivisionFlags = [...]string{
	"\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f", // gbeng
	"\U000e0067\U000e0062\U000e0073\U000e0063\U000e0074\U000e007f", // gbsct
	"\U000e0067\U000e0062\U000e0077\U000e006c\U000e0073\U000e007f", // gbwls
}

// RemoveBadSymbols removes characters that are invisible or reorder text,
// the material of Trojan Source and homoglyph tricks:
//
//   - control characters (Cc), tab and line breaks included;
//   - format characters (Cf), which include the bidirectional controls such
//     as U+202E and U+2066-U+2069, the marks U+200E, U+200F and U+061C, the
//     zero width space, non-joiner and joiner U+200B-U+200D, the byte order
//     mark and the tag characters U+E0001-U+E007F;
//   - the line and paragraph separators U+2028 and U+2029, the object
//     replacement character U+FFFC, the deprecated U+0340, U+0341, U+17A3 and
//     U+17D3, and the unassigned tag U+E0000.
//
// Two emoji constructs keep their format characters: a zero width joiner
// between two emoji, so that sequences such as woman technologist (U+1F469
// U+200D U+1F4BB) survive, and the tags of the three recommended (RGI)
// subdivision flags, England, Scotland and Wales: U+1F3F4 followed by the tag
// letters gbeng, gbsct or gbwls and U+E007F. Every other tag run after U+1F3F4,
// syntactically valid or not, is removed like any format character. An
// emoji here is a symbol of category So, and the rune before the joiner may
// also be U+FE0F or a skin tone modifier. Scripts that need the
// joiners between letters, such as Persian or Devanagari, lose them.
//
// A string made only of removed characters becomes empty; combine the
// modifier with nil_on_empty to clear a pointer instead. Each byte that is not
// valid UTF-8 becomes U+FFFD, so that removing a character never joins
// invalid bytes into a valid one.
func RemoveBadSymbols(v reflect.Value, _ map[string]string) Result {
	return ApplyString(v, removeBadSymbols)
}

func removeBadSymbols(s string) (string, bool) {
	var (
		b       strings.Builder
		changed bool
		prev    rune // the last rune kept
	)

	// edit switches to building a new string from s[:i] at the first change.
	edit := func(i int) {
		if !changed {
			b.Grow(len(s))
			b.WriteString(s[:i])

			changed = true
		}
	}

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		keep, n := s[i:i+size], size

		switch {
		case r == blackFlag:
			n += tagSequenceLength(s[i+size:])
			keep = s[i : i+n]
		case r == zeroWidthJoiner && joinsEmoji(prev, s[i+size:]):
		case r == utf8.RuneError && size == 1:
			// Removing a character could join invalid bytes around it into a
			// valid one, so every invalid byte becomes U+FFFD.
			edit(i)

			keep = string(utf8.RuneError)
		case isBadSymbol(r):
			edit(i)

			i += size

			continue
		}

		if changed {
			b.WriteString(keep)
		}

		prev = r
		i += n
	}

	if !changed {
		return s, false
	}

	return b.String(), true
}

// tagSequenceLength returns the length of the tag characters that open s
// when they are those of a recommended (RGI) subdivision flag, and 0 for any
// other run of tags, which could carry invisible text.
func tagSequenceLength(s string) int {
	for _, tags := range rgiSubdivisionFlags {
		if strings.HasPrefix(s, tags) {
			return len(tags)
		}
	}

	return 0
}

// joinsEmoji reports whether a zero width joiner after prev and before rest
// stands between two emoji.
func joinsEmoji(prev rune, rest string) bool {
	next, _ := utf8.DecodeRuneInString(rest)
	return isEmojiEnd(prev) && isEmoji(next)
}

// isEmoji reports whether r is a symbol of category So. The replacement
// character, which also stands for bytes that are not valid UTF-8 and for
// the end of the string, is none.
func isEmoji(r rune) bool { return r != utf8.RuneError && unicode.Is(unicode.So, r) }

// isEmojiEnd reports whether r can end an emoji that a joiner follows: an
// emoji, the emoji presentation selector or a skin tone modifier.
func isEmojiEnd(r rune) bool {
	return isEmoji(r) || r == variationSelector16 || r >= '\U0001f3fb' && r <= '\U0001f3ff'
}

func isBadSymbol(r rune) bool {
	if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Bidi_Control) {
		return true
	}

	switch r {
	case '\u0340', '\u0341', '\u17a3', '\u17d3', '\u2028', '\u2029', '\ufffc', '\U000e0000':
		return true
	}

	return false
}
