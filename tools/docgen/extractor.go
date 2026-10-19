// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"iter"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// cobraExtractor builds documentation data from cobra command metadata.
type cobraExtractor struct {
	// urlBase is the site path of the CLI reference, with slashes on both ends.
	urlBase string
}

// Values the extractor uses to build page data from the help text.
const (
	// maxDescriptionLen is the longest page description, in runes, including
	// the ellipsis that marks a truncated one.
	maxDescriptionLen = 160
	// ellipsis marks a truncated description.
	ellipsis = "..."
	// indexTitle is the CLI reference index page title.
	indexTitle = "CLI Reference"
	// referenceURL is the site path of the CLI reference.
	referenceURL = "/usage/cli-reference/"
	// paragraphBreak separates paragraphs in help text.
	paragraphBreak = "\n\n"
	// commentPrefix starts an example's title line.
	commentPrefix = "#"
	// indentChars are the characters dedent removes.
	indentChars = " \t"
)

// NewCobraExtractor returns an extractor that links pages under the CLI
// reference section.
//
// Returns:
//   - *cobraExtractor: the extractor.
func NewCobraExtractor() *cobraExtractor {
	return &cobraExtractor{urlBase: referenceURL}
}

// Extract builds the documentation for cmd and every available descendant.
//
// Parameters:
//   - cmd: the command to document.
//   - parentPath: the parent's full path, or cmd's own name for the root.
//
// Returns:
//   - *commandDoc: the documentation data.
func (e *cobraExtractor) Extract(cmd *cobra.Command, parentPath string) *commandDoc {
	fullPath := e.buildFullPath(cmd, parentPath)

	doc := &commandDoc{
		Name:        cmd.Name(),
		Short:       cmd.Short,
		Long:        strings.TrimSpace(cmd.Long),
		Use:         cmd.Use,
		Example:     cmd.Example,
		UseLine:     e.buildUseLine(cmd.Use, fullPath),
		Title:       e.buildTitle(cmd),
		Description: e.buildDescription(cmd.Long),
		FullPath:    fullPath,
		Flags:       nil,
		Inherited:   nil,
		Examples:    nil,
		SubCommands: nil,
		HasSubs:     false,
		Index:       nil,
	}

	if cmd.Example != "" {
		doc.Examples = e.parseExamples(cmd.Example, fullPath, cmd.Use)
	}

	if own := cmd.NonInheritedFlags(); own.HasAvailableFlags() {
		doc.Flags = e.buildFlags(own)
	}

	if cmd.HasAvailableInheritedFlags() {
		doc.Inherited = e.buildFlags(cmd.InheritedFlags())
	}

	if cmd.HasAvailableSubCommands() {
		doc.HasSubs = true
		doc.Index = e.buildIndex(cmd)
	}

	for child := range availableCommands(cmd) {
		doc.SubCommands = append(doc.SubCommands, e.Extract(child, fullPath))
	}

	return doc
}

// buildDescription turns long help text into a one-line page description.
//
// The first paragraph is used, with newlines folded into spaces so it fits a
// frontmatter line. A longer text is cut on a rune boundary and ends with an
// ellipsis.
//
// Parameters:
//   - long: the long help text.
//
// Returns:
//   - string: at most maxDescriptionLen runes.
func (e *cobraExtractor) buildDescription(long string) string {
	desc := firstParagraph(long)
	if utf8.RuneCountInString(desc) <= maxDescriptionLen {
		return desc
	}

	runes := []rune(desc)
	cut := strings.TrimRightFunc(string(runes[:maxDescriptionLen-len(ellipsis)]), unicode.IsSpace)

	return cut + ellipsis
}

// buildFlags lists the visible flags of a flag set in name order.
//
// Parameters:
//   - flags: the flag set.
//
// Returns:
//   - []flagDoc: one entry per flag that is neither hidden nor deprecated.
func (e *cobraExtractor) buildFlags(flags *pflag.FlagSet) []flagDoc {
	var docs []flagDoc

	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden || flag.Deprecated != "" {
			return
		}

		docs = append(docs, flagDoc{
			Name:      flag.Name,
			Shorthand: flag.Shorthand,
			Default:   flag.DefValue,
			Type:      flag.Value.Type(),
			Usage:     flag.Usage,
		})
	})

	return docs
}

// buildFullPath joins the parent path and the command name.
//
// Parameters:
//   - cmd: the command.
//   - parentPath: the parent's full path, or cmd's own name for the root.
//
// Returns:
//   - string: the full path, such as "speedtest-go list".
func (e *cobraExtractor) buildFullPath(cmd *cobra.Command, parentPath string) string {
	if cmd.Name() == parentPath {
		return parentPath
	}

	return parentPath + " " + cmd.Name()
}

// buildIndex builds the index page for a command with child commands.
//
// Parameters:
//   - cmd: the command whose children the index lists.
//
// Returns:
//   - *indexDoc: the index data.
func (e *cobraExtractor) buildIndex(cmd *cobra.Command) *indexDoc {
	index := &indexDoc{
		Title:       indexTitle,
		Description: "Every " + cmd.Name() + " command and flag, generated from the program itself",
		Name:        cmd.Name(),
		Long:        strings.TrimSpace(cmd.Long),
		UseLine:     e.buildUseLine(cmd.Use, cmd.Name()),
		Sections:    nil,
		Examples:    nil,
		Options:     nil,
		Flags:       nil,
	}

	if cmd.Example != "" {
		index.Examples = e.parseExamples(cmd.Example, cmd.CommandPath(), cmd.Use)
	}

	// The root's own flags apply only when it runs, and its persistent flags apply to every command.
	if own := cmd.LocalNonPersistentFlags(); own.HasAvailableFlags() {
		index.Options = e.buildFlags(own)
	}

	if persistent := cmd.PersistentFlags(); persistent.HasAvailableFlags() {
		index.Flags = e.buildFlags(persistent)
	}

	for child := range availableCommands(cmd) {
		index.Sections = append(index.Sections, e.buildSection(child))
	}

	return index
}

// buildSection builds the index entry for one top-level command.
//
// A group lists its children, and a leaf command lists itself.
//
// Parameters:
//   - child: the top-level command.
//
// Returns:
//   - sectionEntry: the entry.
func (e *cobraExtractor) buildSection(child *cobra.Command) sectionEntry {
	section := sectionEntry{
		Name:        child.Name(),
		Description: firstParagraph(child.Long),
		HasSubs:     child.HasAvailableSubCommands(),
		Title:       e.buildTitle(child),
		SubCommands: nil,
	}

	if !section.HasSubs {
		section.SubCommands = []subCommandEntry{{
			Name:        child.Name(),
			Description: child.Short,
			URL:         e.urlBase + child.Name() + "/",
		}}

		return section
	}

	for sub := range availableCommands(child) {
		section.SubCommands = append(section.SubCommands, subCommandEntry{
			Name:        sub.Name(),
			Description: sub.Short,
			URL:         e.urlBase + child.Name() + "/" + sub.Name() + "/",
		})
	}

	return section
}

// buildTitle returns the page title for a command.
//
// Parameters:
//   - cmd: the command.
//
// Returns:
//   - string: the title-cased name of a leaf command, or a group's Short text.
func (e *cobraExtractor) buildTitle(cmd *cobra.Command) string {
	if !cmd.HasAvailableSubCommands() || cmd.Short == "" {
		return titleCase(cmd.Name())
	}

	return cmd.Short
}

// buildUseLine replaces the command name in a Use line with the full path.
//
// Parameters:
//   - use: the cobra Use line.
//   - fullPath: the command's full path.
//
// Returns:
//   - string: the usage line, or empty for an empty Use line.
func (e *cobraExtractor) buildUseLine(use, fullPath string) string {
	if strings.TrimSpace(use) == "" {
		return ""
	}

	args := useArgs(use)
	if args == "" {
		return fullPath
	}

	return fullPath + " " + args
}

// parseExamples splits cobra Example text into titled blocks.
//
// Blocks are separated by lines that are empty or hold only white space.
// Lines starting with "#" form the block's title and the remaining lines its
// code. An example that only repeats the usage line is dropped, because the
// page already shows it.
//
// Parameters:
//   - raw: the cobra Example text.
//   - fullPath: the command's full path.
//   - use: the cobra Use line.
//
// Returns:
//   - []exampleDoc: the examples, or nil when none add anything.
func (e *cobraExtractor) parseExamples(raw, fullPath, use string) []exampleDoc {
	var (
		examples  []exampleDoc
		titles    []string
		codeLines []string
	)

	// flush ends the current block. A block without code adds no example.
	flush := func() {
		if len(codeLines) > 0 {
			examples = append(examples, exampleDoc{
				Title: strings.TrimSpace(strings.Join(titles, " ")),
				Code:  dedent(codeLines),
			})
		}

		titles, codeLines = nil, nil
	}

	for line := range strings.SplitSeq(raw, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			flush()
		case strings.HasPrefix(trimmed, commentPrefix):
			titles = append(titles, strings.TrimSpace(strings.TrimLeft(trimmed, commentPrefix)))
		default:
			codeLines = append(codeLines, strings.TrimRightFunc(line, unicode.IsSpace))
		}
	}

	flush()

	if len(examples) == 1 && examples[0].Code == e.buildUseLine(use, fullPath) {
		return nil
	}

	return examples
}

// availableCommands yields the children of cmd that a user can run.
//
// Hidden, deprecated, and help topic commands are skipped. Cobra returns
// children sorted by name, so the order is stable.
//
// Parameters:
//   - cmd: the parent command.
//
// Returns:
//   - [iter.Seq]: the available children.
func availableCommands(cmd *cobra.Command) iter.Seq[*cobra.Command] {
	return func(yield func(*cobra.Command) bool) {
		for _, child := range cmd.Commands() {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}

			if !yield(child) {
				return
			}
		}
	}
}

// dedent removes the indentation every line shares.
//
// Only spaces and tabs count as indentation, so a cut never splits a rune.
//
// Parameters:
//   - lines: lines with trailing space removed. At least one is not blank.
//
// Returns:
//   - string: the lines joined by newlines.
func dedent(lines []string) string {
	indent := -1

	for _, line := range lines {
		width := len(line) - len(strings.TrimLeft(line, indentChars))
		if indent < 0 || width < indent {
			indent = width
		}
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, line[indent:])
	}

	return strings.Join(out, "\n")
}

// firstParagraph returns the first paragraph of text on one line.
//
// Parameters:
//   - text: the text.
//
// Returns:
//   - string: the first paragraph with runs of white space folded into
//     single spaces.
func firstParagraph(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, paragraphBreak); idx != -1 {
		text = text[:idx]
	}

	return strings.Join(strings.Fields(text), " ")
}

// titleCase upper-cases the first letter of each space-separated word.
//
// Parameters:
//   - text: the text, such as "list-servers".
//
// Returns:
//   - string: the title, such as "List-servers".
func titleCase(text string) string {
	words := strings.Fields(text)
	for i, word := range words {
		first, size := utf8.DecodeRuneInString(word)

		words[i] = string(unicode.ToTitle(first)) + word[size:]
	}

	return strings.Join(words, " ")
}

// useArgs returns the argument part of a cobra Use line.
//
// Parameters:
//   - use: the cobra Use line, such as "status [flags]".
//
// Returns:
//   - string: everything after the command name, trimmed.
func useArgs(use string) string {
	use = strings.TrimSpace(use)

	_, args, _ := strings.Cut(use, " ")

	return strings.TrimSpace(args)
}
