// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// FuzzParseExamples checks example parsing on arbitrary cobra Example text.
//
// Seeds: the shapes the command tree uses (titled blocks, several command
// lines, an untitled usage repeat), indentation, comment-only blocks, and
// text that is only white space.
//
// Invariants: no panic, every example has non-blank code, no code line is
// a comment, no title contains a newline, and parsing is deterministic.
func FuzzParseExamples(f *testing.F) {
	for _, seed := range []string{
		"# Test with TCP.\nspeedtest-go --protocol tcp",
		"# List servers near a city.\nspeedtest-go list --city tokyo\nspeedtest-go --server 6691",
		"speedtest-go list",
		"  # Indented.\n  tool a \\\n    --flag",
		"# Only a comment.\n\n\n#\n",
		" \t\n\n ",
		" x\n\ty",
	} {
		f.Add(seed)
	}

	extractor := NewCobraExtractor()

	f.Fuzz(func(t *testing.T, raw string) {
		examples := extractor.parseExamples(raw, "tool cmd", "cmd")

		for _, example := range examples {
			require.NotEmpty(t, strings.TrimSpace(example.Code))
			assert.NotContains(t, example.Title, "\n")

			for line := range strings.SplitSeq(example.Code, "\n") {
				assert.False(
					t,
					strings.HasPrefix(strings.TrimSpace(line), commentPrefix),
					"comment in code: %q",
					line,
				)
			}
		}

		assert.Equal(t, examples, extractor.parseExamples(raw, "tool cmd", "cmd"))
	})
}

// FuzzBuildDescription checks page descriptions on arbitrary help text.
//
// Seeds: a short sentence, a multi-paragraph text, a text just over the
// limit, and multibyte text over the limit.
//
// Invariants: the result has no newline, never exceeds maxDescriptionLen
// runes, and is valid UTF-8 whenever the input is.
func FuzzBuildDescription(f *testing.F) {
	for _, seed := range []string{
		"Print version",
		"First paragraph\nwraps.\n\nSecond paragraph.",
		strings.Repeat("x", maxDescriptionLen+1),
		strings.Repeat("é ", maxDescriptionLen),
	} {
		f.Add(seed)
	}

	extractor := NewCobraExtractor()

	f.Fuzz(func(t *testing.T, long string) {
		got := extractor.buildDescription(long)

		assert.NotContains(t, got, "\n")
		assert.LessOrEqual(t, utf8.RuneCountInString(got), maxDescriptionLen)

		if utf8.ValidString(long) {
			assert.True(t, utf8.ValidString(got))
		}
	})
}

// FuzzYAMLString checks frontmatter quoting round-trips through a YAML
// parser.
//
// Seeds: plain text, a colon that breaks a plain scalar, quotes and
// backslashes, leading indicator characters, control characters, and
// non-ASCII text.
//
// Invariants: for valid UTF-8, the quoted value decodes to the original
// text as a YAML mapping value.
func FuzzYAMLString(f *testing.F) {
	for _, seed := range []string{
		"List",
		"Select a method for Ping (support icmp/tcp/http).",
		`say "hi" \ bye`,
		"- * & ! | > ' % @ `",
		"tab\there\x00\x1b",
		"é   \U0001F600",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		if !utf8.ValidString(text) {
			t.Skip("frontmatter text is valid UTF-8")
		}

		var decoded map[string]string

		require.NoError(t, yaml.Unmarshal([]byte("description: "+yamlString(text)+"\n"), &decoded))
		assert.Equal(t, text, decoded["description"])
	})
}

// FuzzTableCell checks table cells on arbitrary text.
//
// Seeds: plain text, pipes, escaped pipes, and newlines.
//
// Invariants: the cell has no newline and every pipe is escaped, so it
// cannot end the cell or the row.
func FuzzTableCell(f *testing.F) {
	for _, seed := range []string{"text", "a|b", `a\|b`, "a\n|\nb"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		got := tableCell(text)

		assert.NotContains(t, got, "\n")

		for i := range len(got) {
			if got[i] == '|' {
				require.Positive(t, i)
				assert.Equal(t, byte('\\'), got[i-1], "unescaped pipe in %q", got)
			}
		}
	})
}

// FuzzInlineCode checks code spans on arbitrary text.
//
// Seeds: plain text, inner and edge backticks, and a long backtick run.
//
// Invariants: non-blank text yields a span that opens and closes with the
// same fence, and no backtick run inside is as long as the fence.
func FuzzInlineCode(f *testing.F) {
	for _, seed := range []string{"--mode", "a`b", "`a", "a```b", " "} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		got := inlineCode(text)
		if strings.TrimSpace(strings.Join(strings.Fields(text), " ")) == "" {
			assert.Empty(t, got)

			return
		}

		fenceLen := len(got) - len(strings.TrimLeft(got, backtick))
		fence := strings.Repeat(backtick, fenceLen)

		require.True(t, strings.HasSuffix(got, fence))

		inner := got[fenceLen : len(got)-fenceLen]
		assert.NotContains(t, inner, fence)
		assert.NotContains(t, got, "\n")
	})
}
