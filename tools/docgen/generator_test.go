// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/cmd"
)

// fakeExtractor returns a fixed document for any command.
type fakeExtractor struct {
	// doc is the document Extract returns.
	doc *commandDoc
}

// fakeRenderer records the pages it is asked to write and can fail on one.
type fakeRenderer struct {
	// err is returned for the path that ends with failOn.
	err error
	// failOn is the path suffix that fails, or empty to never fail.
	failOn string
	// paths lists every requested page in order.
	paths []string
}

// errRender is the failure fakeRenderer returns.
var errRender = errors.New("render failed")

// Extract returns the fixed document.
//
// Parameters:
//   - cmd: ignored.
//   - parentPath: ignored.
//
// Returns:
//   - *commandDoc: the fixed document.
func (f *fakeExtractor) Extract(*cobra.Command, string) *commandDoc {
	return f.doc
}

// RenderCommand records path.
//
// Parameters:
//   - doc: ignored.
//   - path: the requested page.
//
// Returns:
//   - error: err when path ends with failOn.
func (f *fakeRenderer) RenderCommand(_ *commandDoc, path string) error {
	return f.record(path)
}

// RenderIndex records path.
//
// Parameters:
//   - doc: ignored.
//   - path: the requested page.
//
// Returns:
//   - error: err when path ends with failOn.
func (f *fakeRenderer) RenderIndex(_ *indexDoc, path string) error {
	return f.record(path)
}

// record stores path and returns the configured failure for it.
//
// Parameters:
//   - path: the requested page.
//
// Returns:
//   - error: err when path ends with failOn.
func (f *fakeRenderer) record(path string) error {
	f.paths = append(f.paths, path)

	if f.failOn != "" && strings.HasSuffix(filepath.ToSlash(path), f.failOn) {
		return f.err
	}

	return nil
}

// sampleDoc returns a root document with a leaf and a group.
//
// Returns:
//   - *commandDoc: the document.
func sampleDoc() *commandDoc {
	return &commandDoc{
		Name:  "root",
		Index: &indexDoc{Title: indexTitle},
		SubCommands: []*commandDoc{
			{Name: "leaf"},
			{Name: "group", SubCommands: []*commandDoc{{Name: "child"}}},
		},
	}
}

// TestNewDocGenerator checks the default generator parses its templates.
func TestNewDocGenerator(t *testing.T) {
	t.Parallel()

	got, err := NewDocGenerator()

	require.NoError(t, err)
	assert.IsType(t, &cobraExtractor{}, got.extractor)
	assert.IsType(t, &hugoRenderer{}, got.renderer)
}

// TestNewDocGeneratorWithDeps checks the collaborators are kept.
func TestNewDocGeneratorWithDeps(t *testing.T) {
	t.Parallel()

	extractor, renderer := &fakeExtractor{doc: nil}, &fakeRenderer{err: nil, failOn: "", paths: nil}

	got := NewDocGeneratorWithDeps(extractor, renderer)

	assert.Same(t, extractor, got.extractor)
	assert.Same(t, renderer, got.renderer)
}

// TestDocGenerator_Generate covers the entry point against the real tree.
//
// Every other generator test injects fakes, so this is the test that proves
// the shipped command tree produces the published pages.
func TestDocGenerator_Generate(t *testing.T) {
	t.Parallel()

	generator, err := NewDocGenerator()
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "cli-reference")

	require.NoError(t, generator.Generate(cmd.DocRoot(), out))

	for _, page := range []string{
		"_index.md",
		"cities/_index.md",
		"list/_index.md",
		"version/_index.md",
	} {
		assert.FileExists(t, filepath.Join(out, page))
	}

	for _, page := range []string{"help/_index.md", "completion/_index.md"} {
		assert.NoFileExists(t, filepath.Join(out, page))
	}

	page, err := os.ReadFile(filepath.Join(out, "list", "_index.md"))
	require.NoError(t, err)

	assert.Contains(t, string(page), "title: \"List\"")
	assert.Contains(t, string(page), "type: docs")
	assert.Contains(t, string(page), "speedtest-go list")
	assert.Contains(t, string(page), "| `--filter-cc` |", "a list page carries the global options")
	assert.NotContains(t, string(page), "| `--json` |", "a list page omits the speed test options")

	index, err := os.ReadFile(filepath.Join(out, "_index.md"))
	require.NoError(t, err)

	options, global, found := strings.Cut(string(index), "## Global options")
	require.True(t, found, "the index lists the global options")
	assert.Contains(
		t,
		options,
		"| `--ping-mode` |  | `http` | string |",
		"the speed test options come first",
	)
	assert.NotContains(t, global, "| `--ping-mode` |", "a speed test option is not global")
	assert.Contains(t, global, "| `--config` |", "a persistent flag is global")
}

// TestDocGenerator_GenerateReportsAnUnusableOutputDirectory checks a failed
// setup is reported instead of writing a partial tree.
func TestDocGenerator_GenerateReportsAnUnusableOutputDirectory(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), filePerms))

	generator, err := NewDocGenerator()
	require.NoError(t, err)

	err = generator.Generate(cmd.DocRoot(), filepath.Join(blocker, "cli"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create output directory")
}

// TestDocGenerator_generateAll covers the page order, a root without
// commands, and render failures.
func TestDocGenerator_generateAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		doc     *commandDoc
		name    string
		failOn  string
		want    []string
	}{
		{
			name: "index first, then each section depth first",
			doc:  sampleDoc(),
			want: []string{
				"_index.md",
				"leaf/_index.md",
				"group/_index.md",
				"group/child/_index.md",
			},
		},
		{
			name:    "root without commands",
			doc:     &commandDoc{Name: "root"},
			wantErr: ErrNoCommands,
		},
		{
			name:    "index failure",
			doc:     sampleDoc(),
			failOn:  "/_index.md",
			wantErr: errRender,
			want:    []string{"_index.md"},
		},
		{
			name:    "nested failure",
			doc:     sampleDoc(),
			failOn:  "child/_index.md",
			wantErr: errRender,
			want: []string{
				"_index.md",
				"leaf/_index.md",
				"group/_index.md",
				"group/child/_index.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := t.TempDir()
			renderer := &fakeRenderer{err: errRender, failOn: tt.failOn, paths: nil}
			generator := NewDocGeneratorWithDeps(&fakeExtractor{doc: tt.doc}, renderer)

			err := generator.generateAll(tt.doc, out)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			var got []string

			for _, path := range renderer.paths {
				rel, relErr := filepath.Rel(out, path)
				require.NoError(t, relErr)

				got = append(got, filepath.ToSlash(rel))
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDocGenerator_generateSection checks a section directory that cannot
// be created is reported.
func TestDocGenerator_generateSection(t *testing.T) {
	t.Parallel()

	out := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(out, "leaf"), nil, filePerms))

	generator := NewDocGeneratorWithDeps(
		&fakeExtractor{doc: nil},
		&fakeRenderer{err: nil, failOn: "", paths: nil},
	)

	err := generator.generateSection(&commandDoc{Name: "leaf"}, out)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create ")
}
