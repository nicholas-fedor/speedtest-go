// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noRun is a command body for test commands, which makes them available.
func noRun(*cobra.Command, []string) {}

// newTestTree builds a small tree with a leaf, a group, a hidden command, and
// flags of every visibility.
//
// Parameters:
//   - t: the test, which fails when the tree cannot be built.
//
// Returns:
//   - *cobra.Command: the root command.
func newTestTree(t *testing.T) *cobra.Command {
	t.Helper()

	root := &cobra.Command{
		Use:     "tool",
		Short:   "A tool",
		Long:    "Tool does things.\n\nIt does them well.",
		Example: "# Run the tool.\ntool run",
		Run:     noRun,
	}
	root.PersistentFlags().String("mode", "fast", "the mode")

	run := &cobra.Command{
		Use:     "run [name]",
		Short:   "Run something",
		Long:    "Run something by name.",
		Example: "tool run [name]",
		Run:     noRun,
	}
	run.Flags().BoolP("dry", "n", false, "print instead")
	run.Flags().String("secret", "", "hidden flag")
	require.NoError(t, run.Flags().MarkHidden("secret"))

	group := &cobra.Command{Use: "config", Short: "Manage config", Long: "Manage config."}
	group.AddCommand(
		&cobra.Command{Use: "show", Short: "Show config", Run: noRun},
		&cobra.Command{Use: "edit", Short: "Edit config", Run: noRun},
	)

	hidden := &cobra.Command{Use: "internal", Short: "Hidden", Hidden: true, Run: noRun}

	root.AddCommand(run, group, hidden)

	return root
}

// TestNewCobraExtractor checks the extractor links pages under the CLI
// reference section.
func TestNewCobraExtractor(t *testing.T) {
	t.Parallel()

	got := NewCobraExtractor()

	require.NotNil(t, got)
	assert.Equal(t, referenceURL, got.urlBase)
}

// Test_cobraExtractor_Extract covers a whole tree: available commands only,
// in name order, with own and inherited flags kept apart.
func Test_cobraExtractor_Extract(t *testing.T) {
	t.Parallel()

	doc := NewCobraExtractor().Extract(newTestTree(t), "tool")

	require.NotNil(t, doc)
	assert.Equal(t, "tool", doc.FullPath)
	assert.True(t, doc.HasSubs)
	require.NotNil(t, doc.Index)
	require.Len(t, doc.SubCommands, 2, "the hidden command is skipped")
	assert.Equal(t, "config", doc.SubCommands[0].Name)
	assert.Equal(t, "run", doc.SubCommands[1].Name)

	run := doc.SubCommands[1]
	assert.Equal(t, "tool run", run.FullPath)
	assert.Equal(t, "tool run [name]", run.UseLine)
	assert.Equal(t, "Run", run.Title)
	assert.Equal(t, "Run something by name.", run.Description)
	assert.Nil(t, run.Examples, "an example that repeats the usage line is dropped")
	require.Len(t, run.Flags, 1, "the hidden flag is skipped")
	assert.Equal(
		t,
		flagDoc{
			Name:      "dry",
			Shorthand: "n",
			Default:   "false",
			Type:      "bool",
			Usage:     "print instead",
		},
		run.Flags[0],
	)
	require.Len(t, run.Inherited, 1)
	assert.Equal(t, "mode", run.Inherited[0].Name)
	assert.False(t, run.HasSubs)
	assert.Nil(t, run.Index)

	group := doc.SubCommands[0]
	assert.Equal(t, "Manage config", group.Title)
	require.Len(t, group.SubCommands, 2)
	assert.Equal(t, "tool config edit", group.SubCommands[0].FullPath)
}

// Test_cobraExtractor_buildDescription covers folding and rune-safe
// truncation of page descriptions.
func Test_cobraExtractor_buildDescription(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("word ", 50)
	runes := strings.Repeat("é", maxDescriptionLen+5)

	tests := []struct {
		name string
		long string
		want string
	}{
		{name: "short text is kept", long: "Short description", want: "Short description"},
		{name: "newlines fold into spaces", long: "Line one\nline two", want: "Line one line two"},
		{name: "only the first paragraph", long: "First.\n\nSecond.", want: "First."},
		{name: "empty text", long: "", want: ""},
		{
			name: "long text is truncated at a word gap",
			long: long,
			want: strings.TrimSpace(long[:maxDescriptionLen-len(ellipsis)]) + ellipsis,
		},
		{
			name: "multibyte text is cut on a rune boundary",
			long: runes,
			want: strings.Repeat("é", maxDescriptionLen-len(ellipsis)) + ellipsis,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := (&cobraExtractor{urlBase: referenceURL}).buildDescription(tt.long)

			assert.Equal(t, tt.want, got)
			assert.True(t, utf8.ValidString(got))
			assert.LessOrEqual(t, utf8.RuneCountInString(got), maxDescriptionLen)
		})
	}
}

// Test_cobraExtractor_buildFlags covers which flags are listed.
func Test_cobraExtractor_buildFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flags func() *pflag.FlagSet
		name  string
		want  []string
	}{
		{
			name:  "no flags",
			flags: func() *pflag.FlagSet { return pflag.NewFlagSet("test", pflag.ContinueOnError) },
			want:  nil,
		},
		{
			name: "sorted by name, hidden and deprecated skipped",
			flags: func() *pflag.FlagSet {
				fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
				fs.String("zeta", "", "last")
				fs.String("alpha", "", "first")
				fs.String("hidden", "", "hidden")
				fs.String("old", "", "deprecated")
				_ = fs.MarkHidden("hidden")
				_ = fs.MarkDeprecated("old", "use alpha")

				return fs
			},
			want: []string{"alpha", "zeta"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, flag := range (&cobraExtractor{urlBase: referenceURL}).buildFlags(tt.flags()) {
				got = append(got, flag.Name)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_cobraExtractor_buildFullPath covers the root and child paths.
func Test_cobraExtractor_buildFullPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		use        string
		parentPath string
		want       string
	}{
		{name: "root keeps its name", use: "root", parentPath: "root", want: "root"},
		{name: "child appends its name", use: "sub", parentPath: "root", want: "root sub"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := (&cobraExtractor{urlBase: referenceURL}).buildFullPath(
				&cobra.Command{Use: tt.use},
				tt.parentPath,
			)

			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_cobraExtractor_buildIndex covers the index for a root with a leaf and
// a group, including the root's own flags and examples.
func Test_cobraExtractor_buildIndex(t *testing.T) {
	t.Parallel()

	index := NewCobraExtractor().buildIndex(newTestTree(t))

	assert.Equal(t, indexTitle, index.Title)
	assert.Equal(
		t,
		"Every tool command and flag, generated from the program itself",
		index.Description,
	)
	assert.Equal(t, "tool", index.Name)
	assert.Equal(t, "Tool does things.\n\nIt does them well.", index.Long)
	assert.Equal(t, []exampleDoc{{Title: "Run the tool.", Code: "tool run"}}, index.Examples)
	require.Len(t, index.Flags, 1)
	assert.Equal(t, "mode", index.Flags[0].Name)

	require.Len(t, index.Sections, 2)
	assert.Equal(t, sectionEntry{
		Name:        "config",
		Title:       "Manage config",
		Description: "Manage config.",
		HasSubs:     true,
		SubCommands: []subCommandEntry{
			{Name: "edit", Description: "Edit config", URL: "/usage/cli-reference/config/edit/"},
			{Name: "show", Description: "Show config", URL: "/usage/cli-reference/config/show/"},
		},
	}, index.Sections[0])
	assert.Equal(t, sectionEntry{
		Name:        "run",
		Title:       "Run",
		Description: "Run something by name.",
		HasSubs:     false,
		SubCommands: []subCommandEntry{
			{Name: "run", Description: "Run something", URL: "/usage/cli-reference/run/"},
		},
	}, index.Sections[1])
}

// Test_cobraExtractor_buildTitle covers leaf and group titles.
func Test_cobraExtractor_buildTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		short     string
		wantTitle string
		hasSubs   bool
	}{
		{name: "leaf command", short: "My Group", hasSubs: false, wantTitle: "Test"},
		{name: "command group", short: "My Group", hasSubs: true, wantTitle: "My Group"},
		{name: "group without short text", short: "", hasSubs: true, wantTitle: "Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test", Short: tt.short}
			if tt.hasSubs {
				cmd.AddCommand(&cobra.Command{Use: "sub", Run: noRun})
			}

			assert.Equal(t, tt.wantTitle, (&cobraExtractor{urlBase: referenceURL}).buildTitle(cmd))
		})
	}
}

// Test_cobraExtractor_buildUseLine covers usage lines with and without
// arguments.
func Test_cobraExtractor_buildUseLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		use      string
		fullPath string
		want     string
	}{
		{name: "empty use", use: "", fullPath: "root", want: ""},
		{name: "blank use", use: "  ", fullPath: "root", want: ""},
		{name: "simple use", use: "test", fullPath: "root test", want: "root test"},
		{
			name:     "use with args",
			use:      "test <name>",
			fullPath: "root test",
			want:     "root test <name>",
		},
		{
			name:     "extra spaces",
			use:      " test   [flags] ",
			fullPath: "root test",
			want:     "root test [flags]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := (&cobraExtractor{urlBase: referenceURL}).buildUseLine(tt.use, tt.fullPath)

			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_cobraExtractor_parseExamples covers titles, multi-line code,
// indentation, and dropping an example that only repeats the usage line.
func Test_cobraExtractor_parseExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []exampleDoc
	}{
		{
			name: "titled blocks",
			raw:  "# First.\ntool a\n\n# Second.\ntool b\ntool c",
			want: []exampleDoc{
				{Title: "First.", Code: "tool a"},
				{Title: "Second.", Code: "tool b\ntool c"},
			},
		},
		{
			name: "untitled block",
			raw:  "tool a --flag",
			want: []exampleDoc{{Title: "", Code: "tool a --flag"}},
		},
		{
			name: "comment lines join into one title",
			raw:  "# Part one,\n## part two.\ntool a",
			want: []exampleDoc{{Title: "Part one, part two.", Code: "tool a"}},
		},
		{
			name: "shared indentation is removed",
			raw:  "  # Indented.\n  tool a \\\n    --flag",
			want: []exampleDoc{{Title: "Indented.", Code: "tool a \\\n  --flag"}},
		},
		{
			name: "comment-only block is skipped",
			raw:  "# Nothing to run.\n\n# Run.\ntool a",
			want: []exampleDoc{{Title: "Run.", Code: "tool a"}},
		},
		{
			name: "usage repeat is dropped",
			raw:  "tool cmd",
			want: nil,
		},
		{
			name: "empty text",
			raw:  "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := (&cobraExtractor{urlBase: referenceURL}).parseExamples(tt.raw, "tool cmd", "cmd")

			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_availableCommands checks iteration skips unavailable commands and
// stops when the consumer does.
func Test_availableCommands(t *testing.T) {
	t.Parallel()

	root := newTestTree(t)

	var names []string
	for child := range availableCommands(root) {
		names = append(names, child.Name())
	}

	assert.Equal(t, []string{"config", "run"}, names)

	var first []string
	for child := range availableCommands(root) {
		first = append(first, child.Name())

		break
	}

	assert.Equal(t, []string{"config"}, first)
}

// Test_dedent covers mixed indentation widths.
func Test_dedent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		lines []string
	}{
		{name: "no indentation", lines: []string{"a", "b"}, want: "a\nb"},
		{name: "common indentation", lines: []string{"  a", "    b"}, want: "a\n  b"},
		{name: "tabs", lines: []string{"\ta", "\t\tb"}, want: "a\n\tb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, dedent(tt.lines))
		})
	}
}

// Test_firstParagraph covers paragraph splitting and space folding.
func Test_firstParagraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "single paragraph", text: "First paragraph", want: "First paragraph"},
		{
			name: "multiple paragraphs",
			text: "First paragraph\n\nSecond paragraph",
			want: "First paragraph",
		},
		{name: "with newlines", text: "Line one\nLine two", want: "Line one Line two"},
		{name: "leading blank lines", text: "\n\nText", want: "Text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, firstParagraph(tt.text))
		})
	}
}

// Test_titleCase covers single words, hyphenated names, and several words.
func Test_titleCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "lowercase", text: "test", want: "Test"},
		{name: "already title case", text: "Test", want: "Test"},
		{name: "hyphenated name", text: "list-servers", want: "List-servers"},
		{name: "multiple words", text: "my command", want: "My Command"},
		{name: "empty", text: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, titleCase(tt.text))
		})
	}
}

// Test_useArgs covers Use lines with and without arguments.
func Test_useArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		use  string
		want string
	}{
		{name: "name only", use: "status", want: ""},
		{name: "with args", use: "get <key> [flags]", want: "<key> [flags]"},
		{name: "surrounding space", use: "  get  <key> ", want: "<key>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, useArgs(tt.use))
		})
	}
}
