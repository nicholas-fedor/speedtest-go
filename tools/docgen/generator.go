// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// DocExtractor builds documentation data from a cobra command tree.
type DocExtractor interface {
	// Extract builds the documentation for cmd and its descendants.
	//
	// Parameters:
	//   - cmd: the command to document.
	//   - parentPath: the parent's full path, or cmd's own name for the root.
	//
	// Returns:
	//   - *commandDoc: the documentation data.
	Extract(cmd *cobra.Command, parentPath string) *commandDoc
}

// TemplateRenderer writes documentation pages.
type TemplateRenderer interface {
	// RenderCommand writes one command's page.
	//
	// Parameters:
	//   - doc: the command's documentation data.
	//   - path: the output file.
	//
	// Returns:
	//   - error: a template or file error.
	RenderCommand(doc *commandDoc, path string) error

	// RenderIndex writes the index page.
	//
	// Parameters:
	//   - doc: the index data.
	//   - path: the output file.
	//
	// Returns:
	//   - error: a template or file error.
	RenderIndex(doc *indexDoc, path string) error
}

// DocGenerator writes a page for every command in a tree.
type DocGenerator struct {
	// extractor builds documentation data from the tree.
	extractor DocExtractor
	// renderer writes the pages.
	renderer TemplateRenderer
}

const (
	// dirPerms is the mode of created directories.
	dirPerms = 0o750
	// pageName is the file name of every generated page.
	pageName = "_index.md"
)

// ErrNoCommands indicates a root command without available subcommands,
// which leaves the index page with nothing to list.
var ErrNoCommands = errors.New("root command has no available subcommands")

// NewDocGenerator returns a generator with the cobra extractor and the Hugo
// renderer.
//
// Returns:
//   - *DocGenerator: the generator.
//   - error: a template parse error.
func NewDocGenerator() (*DocGenerator, error) {
	renderer, err := NewHugoRenderer()
	if err != nil {
		return nil, fmt.Errorf("create renderer: %w", err)
	}

	return NewDocGeneratorWithDeps(NewCobraExtractor(), renderer), nil
}

// NewDocGeneratorWithDeps returns a generator with the given collaborators.
//
// Parameters:
//   - extractor: builds documentation data.
//   - renderer: writes the pages.
//
// Returns:
//   - *DocGenerator: the generator.
func NewDocGeneratorWithDeps(extractor DocExtractor, renderer TemplateRenderer) *DocGenerator {
	return &DocGenerator{
		extractor: extractor,
		renderer:  renderer,
	}
}

// Generate writes the index page and one page per command under out.
//
// Parameters:
//   - root: the root command.
//   - out: the output directory, created when missing.
//
// Returns:
//   - error: ErrNoCommands, or a directory or rendering error.
func (g *DocGenerator) Generate(root *cobra.Command, out string) error {
	err := os.MkdirAll(out, dirPerms)
	if err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	return g.generateAll(g.extractor.Extract(root, root.Name()), out)
}

// generateAll writes the index page and every section.
//
// Parameters:
//   - doc: the root command's documentation data.
//   - out: the output directory.
//
// Returns:
//   - error: ErrNoCommands, or a directory or rendering error.
func (g *DocGenerator) generateAll(doc *commandDoc, out string) error {
	if doc.Index == nil {
		return fmt.Errorf("%w: %s", ErrNoCommands, doc.Name)
	}

	err := g.renderer.RenderIndex(doc.Index, filepath.Join(out, pageName))
	if err != nil {
		return fmt.Errorf("render index: %w", err)
	}

	for _, section := range doc.SubCommands {
		err = g.generateSection(section, out)
		if err != nil {
			return err
		}
	}

	return nil
}

// generateSection writes one command's page and those of its children.
//
// Parameters:
//   - section: the command's documentation data.
//   - out: the directory that receives the command's directory.
//
// Returns:
//   - error: a directory or rendering error.
func (g *DocGenerator) generateSection(section *commandDoc, out string) error {
	sectionDir := filepath.Join(out, section.Name)

	err := os.MkdirAll(sectionDir, dirPerms)
	if err != nil {
		return fmt.Errorf("create %s: %w", sectionDir, err)
	}

	err = g.renderer.RenderCommand(section, filepath.Join(sectionDir, pageName))
	if err != nil {
		return fmt.Errorf("render command %s: %w", section.Name, err)
	}

	for _, sub := range section.SubCommands {
		err = g.generateSection(sub, sectionDir)
		if err != nil {
			return err
		}
	}

	return nil
}
