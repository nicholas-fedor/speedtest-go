// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

// commandDoc is one command's documentation data for template rendering.
//
// Fields stay exported because text/template can only read exported fields.
type commandDoc struct {
	// Index is the command group index, or nil for a leaf command.
	Index *indexDoc
	// Title is the page title.
	Title string
	// FullPath is the command path from the root, such as "speedtest-go list".
	FullPath string
	// Use is the cobra Use line.
	Use string
	// Example is the raw cobra Example text.
	Example string
	// UseLine is the usage line shown on the page.
	UseLine string
	// Name is the command name.
	Name string
	// Description is the one-line page description for the frontmatter.
	Description string
	// Long is the long help text.
	Long string
	// Short is the one-line help text.
	Short string
	// Inherited lists the flags inherited from parent commands.
	Inherited []flagDoc
	// Examples lists the parsed examples.
	Examples []exampleDoc
	// SubCommands lists the available child commands.
	SubCommands []*commandDoc
	// Flags lists the command's own flags.
	Flags []flagDoc
	// HasSubs reports whether the command has available child commands.
	HasSubs bool
}

// flagDoc is one flag's documentation data.
type flagDoc struct {
	// Name is the flag name.
	Name string
	// Shorthand is the single-character shorthand.
	Shorthand string
	// Default is the default value as a string.
	Default string
	// Type is the flag value type.
	Type string
	// Usage is the usage description.
	Usage string
}

// exampleDoc is one example block's documentation data.
type exampleDoc struct {
	// Title is the example title, from its leading comment.
	Title string
	// Code is the example command lines.
	Code string
}

// indexDoc is the CLI reference index page's documentation data.
type indexDoc struct {
	// Title is the index page title.
	Title string
	// Description is the one-line page description for the frontmatter.
	Description string
	// Name is the root command name.
	Name string
	// Long is the root command's long help text.
	Long string
	// UseLine is the root command's usage line.
	UseLine string
	// Sections lists the top-level commands.
	Sections []sectionEntry
	// Examples lists the root command's parsed examples.
	Examples []exampleDoc
	// Options lists the flags only the root command accepts.
	Options []flagDoc
	// Flags lists the root command's persistent flags, which every command accepts.
	Flags []flagDoc
}

// sectionEntry is one top-level command in the index page.
type sectionEntry struct {
	// Name is the command name.
	Name string
	// Title is the section title.
	Title string
	// Description is the first paragraph of the long help text.
	Description string
	// SubCommands lists the entries the index links to.
	SubCommands []subCommandEntry
	// HasSubs reports whether the command is a group with child commands.
	HasSubs bool
}

// subCommandEntry is one linked entry in an index section.
type subCommandEntry struct {
	// Name is the command name.
	Name string
	// Description is the one-line help text.
	Description string
	// URL is the page's site path.
	URL string
}
