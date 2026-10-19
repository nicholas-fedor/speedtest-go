// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// hugoRenderer writes Hugo Markdown pages with YAML frontmatter.
type hugoRenderer struct {
	// commandTmpl renders a command page.
	commandTmpl *template.Template
	// indexTmpl renders the index page.
	indexTmpl *template.Template
}

// Output mode and template names of the rendered pages.
const (
	// filePerms is the mode of written pages.
	filePerms = 0o600
	// commandTemplate is the command page template's file name.
	commandTemplate = "command.tmpl"
	// indexTemplate is the index page template's file name.
	indexTemplate = "index.tmpl"
	// backtick delimits Markdown inline code.
	backtick = "`"
)

// templateFS holds the page templates, so docgen runs from any directory.
//
//go:embed templates/*.tmpl
var templateFS embed.FS

// NewHugoRenderer parses the embedded page templates.
//
// Returns:
//   - *hugoRenderer: the renderer.
//   - error: a template parse error.
func NewHugoRenderer() (*hugoRenderer, error) {
	commandTmpl, err := parseTemplate(commandTemplate)
	if err != nil {
		return nil, err
	}

	indexTmpl, err := parseTemplate(indexTemplate)
	if err != nil {
		return nil, err
	}

	return &hugoRenderer{
		commandTmpl: commandTmpl,
		indexTmpl:   indexTmpl,
	}, nil
}

// RenderCommand writes one command's page.
//
// Parameters:
//   - doc: the command's documentation data.
//   - path: the output file.
//
// Returns:
//   - error: a template or file error.
func (r *hugoRenderer) RenderCommand(doc *commandDoc, path string) error {
	return r.renderTemplate(r.commandTmpl, doc, path)
}

// RenderIndex writes the index page.
//
// Parameters:
//   - doc: the index data.
//   - path: the output file.
//
// Returns:
//   - error: a template or file error.
func (r *hugoRenderer) RenderIndex(doc *indexDoc, path string) error {
	return r.renderTemplate(r.indexTmpl, doc, path)
}

// renderTemplate executes a template and writes the result.
//
// The page is rendered completely before the file is touched, so a template
// error never leaves a partial page.
//
// Parameters:
//   - tmpl: the template.
//   - data: the template data.
//   - path: the output file.
//
// Returns:
//   - error: a template or file error.
func (r *hugoRenderer) renderTemplate(tmpl *template.Template, data any, path string) error {
	var buf bytes.Buffer

	err := tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("execute template for %s: %w", path, err)
	}

	err = os.WriteFile(path, buf.Bytes(), filePerms)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// parseTemplate parses one embedded template with the page helpers.
//
// Parameters:
//   - name: the template's file name in the templates directory.
//
// Returns:
//   - *[template.Template]: the parsed template.
//   - error: a parse error.
func parseTemplate(name string) (*template.Template, error) {
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"cell":       tableCell,
		"code":       inlineCode,
		"defaultVal": defaultValue,
		"yaml":       yamlString,
	}).ParseFS(templateFS, "templates/"+name)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}

	return tmpl, nil
}

// defaultValue returns a flag's default as Markdown inline code.
//
// Parameters:
//   - flag: the flag.
//
// Returns:
//   - string: the default, with an empty string shown as "".
func defaultValue(flag flagDoc) string {
	if flag.Default == "" {
		if flag.Type == "string" {
			return inlineCode(`""`)
		}

		return ""
	}

	return inlineCode(flag.Default)
}

// inlineCode wraps text in a Markdown code span.
//
// The fence is one backtick longer than the longest backtick run inside the
// text, and padding spaces keep a leading or trailing backtick apart from it.
// Newlines become spaces, because a code span is one line.
//
// Parameters:
//   - text: the text.
//
// Returns:
//   - string: the code span, or empty for empty text.
func inlineCode(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}

	longest, run := 0, 0

	for _, char := range text {
		if char != '`' {
			run = 0

			continue
		}

		run++

		longest = max(longest, run)
	}

	fence := strings.Repeat(backtick, longest+1)
	if strings.HasPrefix(text, backtick) || strings.HasSuffix(text, backtick) {
		text = " " + text + " "
	}

	return fence + text + fence
}

// tableCell makes text safe for one Markdown table cell.
//
// Newlines become spaces and pipes are escaped, so the text cannot end the
// cell or the row.
//
// Parameters:
//   - text: the text.
//
// Returns:
//   - string: the cell content.
func tableCell(text string) string {
	text = strings.Join(strings.Fields(text), " ")

	return strings.ReplaceAll(text, "|", `\|`)
}

// yamlString quotes text as a YAML double-quoted scalar.
//
// Every escape [strconv.Quote] writes is also a YAML escape, so text
// containing colons, quotes, or a leading indicator character stays one
// plain string.
//
// Parameters:
//   - text: the text.
//
// Returns:
//   - string: the quoted scalar.
func yamlString(text string) string {
	return strconv.Quote(text)
}
