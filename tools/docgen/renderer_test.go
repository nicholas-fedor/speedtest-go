// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHugoRenderer checks both embedded templates parse.
func TestNewHugoRenderer(t *testing.T) {
	t.Parallel()

	got, err := NewHugoRenderer()

	require.NoError(t, err)
	assert.NotNil(t, got.commandTmpl)
	assert.NotNil(t, got.indexTmpl)
}

// Test_hugoRenderer_RenderCommand checks a full command page, including
// quoting a description that YAML would otherwise misread.
func Test_hugoRenderer_RenderCommand(t *testing.T) {
	t.Parallel()

	r, err := NewHugoRenderer()
	require.NoError(t, err)

	doc := &commandDoc{
		Name:        "get",
		Title:       "Get",
		Description: "Read a value: the first one",
		Long:        "Read a value.",
		UseLine:     "tool get <key>",
		Examples: []exampleDoc{
			{Title: "Read one.", Code: "tool get a"},
			{Title: "", Code: "tool get b"},
		},
		SubCommands: []*commandDoc{{Name: "sub", Short: "A | B"}},
		Flags: []flagDoc{
			{Name: "out", Shorthand: "o", Default: "", Type: "string", Usage: "output file"},
		},
		Inherited: []flagDoc{
			{Name: "verbose", Shorthand: "", Default: "false", Type: "bool", Usage: "log more"},
		},
	}
	path := filepath.Join(t.TempDir(), "_index.md")

	require.NoError(t, r.RenderCommand(doc, path))

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	want := "---\n" +
		"title: \"Get\"\n" +
		"description: \"Read a value: the first one\"\n" +
		"type: docs\n" +
		"---\n\n" +
		"Read a value.\n\n" +
		"## Usage\n\n```bash\ntool get <key>\n```\n\n" +
		"## Examples\n\n### Read one.\n\n```bash\ntool get a\n```\n\n```bash\ntool get b\n```\n\n" +
		"## Commands\n\n| Command | Description |\n| --- | --- |\n| [sub](sub/) | A \\| B |\n\n" +
		"## Options\n\n| Flag | Short | Default | Type | Description |\n| --- | --- | --- | --- | --- |\n" +
		"| `--out` | `-o` | `\"\"` | string | output file |\n\n" +
		"## Global options\n\n"

	assert.Contains(t, string(content), want)
	assert.Contains(t, string(content), "| `--verbose` |  | `false` | bool | log more |\n")
}

// Test_hugoRenderer_RenderIndex checks the index lists every entry and the
// root's flags.
func Test_hugoRenderer_RenderIndex(t *testing.T) {
	t.Parallel()

	r, err := NewHugoRenderer()
	require.NoError(t, err)

	doc := &indexDoc{
		Title:       indexTitle,
		Description: "Every tool command",
		Name:        "tool",
		Long:        "Tool does things.",
		Sections: []sectionEntry{
			{
				Name: "run",
				SubCommands: []subCommandEntry{
					{Name: "run", Description: "Run it", URL: "/cli-reference/run/"},
				},
			},
		},
		Examples: []exampleDoc{{Title: "Run.", Code: "tool run"}},
		Flags:    []flagDoc{{Name: "mode", Default: "fast", Type: "string", Usage: "the mode"}},
	}
	path := filepath.Join(t.TempDir(), "_index.md")

	require.NoError(t, r.RenderIndex(doc, path))

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	page := string(content)
	assert.Contains(t, page, "title: \"CLI Reference\"\n")
	assert.Contains(t, page, "Complete command reference for tool")
	assert.Contains(t, page, "Tool does things.")
	assert.Contains(t, page, "| [run](/cli-reference/run/) | Run it |\n")
	assert.Contains(t, page, "### Run.\n\n```bash\ntool run\n```\n")
	assert.Contains(t, page, "| `--mode` |  | `fast` | string | the mode |\n")
}

// Test_hugoRenderer_renderTemplate covers template and write failures.
func Test_hugoRenderer_renderTemplate(t *testing.T) {
	t.Parallel()

	failing := template.Must(template.New("bad").Parse("{{ .Missing }}"))

	tests := []struct {
		tmpl    *template.Template
		name    string
		path    func(t *testing.T) string
		wantErr string
	}{
		{
			name: "template error leaves no file",
			tmpl: failing,
			path: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "page.md")
			},
			wantErr: "execute template",
		},
		{
			name: "write error",
			tmpl: template.Must(template.New("ok").Parse("ok")),
			path: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "missing", "page.md")
			},
			wantErr: "write ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.path(t)

			err := (&hugoRenderer{commandTmpl: nil, indexTmpl: nil}).renderTemplate(
				tt.tmpl,
				struct{}{},
				path,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.NoFileExists(t, path)
		})
	}
}

// Test_parseTemplate covers a template that is not embedded.
func Test_parseTemplate(t *testing.T) {
	t.Parallel()

	_, err := parseTemplate("missing.tmpl")

	require.Error(t, err)
}

// Test_defaultValue covers empty and set defaults of each kind.
func Test_defaultValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		flag flagDoc
		want string
	}{
		{name: "empty string", flag: flagDoc{Type: "string", Default: ""}, want: "`\"\"`"},
		{name: "empty other type", flag: flagDoc{Type: "stringSlice", Default: ""}, want: ""},
		{name: "duration", flag: flagDoc{Type: "duration", Default: "5m0s"}, want: "`5m0s`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, defaultValue(tt.flag))
		})
	}
}

// Test_inlineCode covers fences around text that holds backticks.
func Test_inlineCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: "", want: ""},
		{name: "plain", text: "--mode", want: "`--mode`"},
		{name: "inner backtick", text: "a`b", want: "``a`b``"},
		{name: "backtick run", text: "a``b", want: "```a``b```"},
		{name: "edge backtick", text: "`a", want: "`` `a ``"},
		{name: "newline folds", text: "a\nb", want: "`a b`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, inlineCode(tt.text))
		})
	}
}

// Test_tableCell covers pipes and newlines.
func Test_tableCell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "plain", text: "text", want: "text"},
		{name: "pipe", text: "a|b", want: `a\|b`},
		{name: "newline", text: "a\n\nb", want: "a b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tableCell(tt.text))
		})
	}
}

// Test_yamlString covers text YAML would misread unquoted.
func Test_yamlString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "plain", text: "Status", want: `"Status"`},
		{name: "colon", text: "a: b", want: `"a: b"`},
		{name: "quote", text: `say "hi"`, want: `"say \"hi\""`},
		{name: "html stays literal", text: "<a & b>", want: `"<a & b>"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, yamlString(tt.text))
		})
	}
}
