// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"unicode/utf8"

	"go.yaml.in/yaml/v4"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/flags"
)

// File is the content of the YAML settings file. Its keys are the flag names in camelCase.
//
// Every field is a pointer, so an absent key leaves the setting to its default.
type File struct {
	Proxy         *string   `yaml:"proxy"`
	Source        *string   `yaml:"source"`
	DNSBindSource *bool     `yaml:"dnsBindSource"`
	UserAgent     *string   `yaml:"ua"`
	Location      *string   `yaml:"location"`
	City          *string   `yaml:"city"`
	Search        *string   `yaml:"search"`
	FilterCC      *[]string `yaml:"filterCC"`
	DSCP          *int      `yaml:"dscp"`
	MaskISP       *bool     `yaml:"maskISP"`
	Debug         *bool     `yaml:"debug"`
	BaseURL       *string   `yaml:"baseURL"`
	Server        *[]int    `yaml:"server"`
	CustomURL     *string   `yaml:"customURL"`
	SavingMode    *bool     `yaml:"savingMode"`
	JSON          *bool     `yaml:"json"`
	JSONL         *bool     `yaml:"jsonl"`
	Unix          *bool     `yaml:"unix"`
	Multi         *bool     `yaml:"multi"`
	Thread        *int      `yaml:"thread"`
	NoDownload    *bool     `yaml:"noDownload"`
	NoUpload      *bool     `yaml:"noUpload"`
	PingMode      *string   `yaml:"pingMode"`
	Protocol      *string   `yaml:"protocol"`
	DownloadSize  *int      `yaml:"dlSize"`
	UploadSize    *int      `yaml:"ulSize"`
	Unit          *string   `yaml:"unit"`
}

// fileValue is a setting's value from the settings file.
type fileValue struct {
	// key is the setting's key in the file.
	key string
	// text is a single value.
	text string
	// list holds the entries of a list value.
	list []string
}

// maxFileSize bounds the settings file. Real files are a few hundred bytes.
const maxFileSize = 64 << 10

// Settings file errors.
var (
	// ErrInvalidFile wraps a settings file that cannot be read or decoded.
	ErrInvalidFile = errors.New("invalid settings file")
	// errFileTooLarge reports a settings file above maxFileSize.
	errFileTooLarge = errors.New("settings file is too large")
	// errMultipleDocuments reports a settings file with more than one YAML document.
	errMultipleDocuments = errors.New("settings file holds more than one document")
	// errNotUTF8 reports a settings file that is not UTF-8 text.
	errNotUTF8 = errors.New("settings file is not UTF-8 text")
	// errNotMapping reports a settings document that is not a mapping of setting names.
	errNotMapping = errors.New("settings file is not a mapping of setting names")
	// errUnsupportedYAML reports YAML anchors, aliases, or merge keys.
	errUnsupportedYAML = errors.New("anchors, aliases, and merge keys are not supported")
)

// LoadFile reads and strictly decodes a settings file that must exist.
//
// Unknown keys, repeated keys, and values of the wrong type are errors, so a typo never passes silently. An empty
// file holds no settings.
//
// Parameters:
//   - path: the file to read.
//
// Returns:
//   - File: the decoded settings.
//   - error: an error wrapping ErrInvalidFile.
func LoadFile(path string) (File, error) {
	data, err := readFile(path)
	if err != nil {
		return File{}, fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	file, err := DecodeFile(data)
	if err != nil {
		return File{}, fmt.Errorf("%s: %w", path, err)
	}

	return file, nil
}

// LoadOptionalFile reads a settings file like LoadFile, but a missing file holds no settings.
//
// Parameters:
//   - path: the file to read.
//
// Returns:
//   - File: the decoded settings.
//   - error: an error wrapping ErrInvalidFile.
func LoadOptionalFile(path string) (File, error) {
	file, err := LoadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, nil
	}

	return file, err
}

// DecodeFile strictly decodes settings file content.
//
// Content without a document, such as a file of comments, holds no settings, and so does a document holding only a
// null. Otherwise the document must be a mapping of setting names. A second document, YAML anchors, aliases, merge
// keys, and content that is not UTF-8 are errors.
//
// Parameters:
//   - data: the YAML content.
//
// Returns:
//   - File: the decoded settings.
//   - error: an error wrapping ErrInvalidFile.
func DecodeFile(data []byte) (File, error) {
	var file File

	doc, err := parseDocument(data)
	if err != nil || doc == nil || isNull(doc.Content[0]) {
		return file, err
	}

	err = doc.Content[0].Load(&file, yaml.WithKnownFields())
	if err != nil {
		return File{}, fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	return file, nil
}

// values returns the settings the file sets, keyed by flag name.
//
// Returns:
//   - map[string]fileValue: the values, each with its key in the file.
func (f File) values() map[string]fileValue {
	values := make(map[string]fileValue)

	text := func(name, key string, value *string) {
		if value != nil {
			values[name] = fileValue{key: key, text: *value, list: nil}
		}
	}

	boolean := func(name, key string, value *bool) {
		if value != nil {
			values[name] = fileValue{key: key, text: strconv.FormatBool(*value), list: nil}
		}
	}

	number := func(name, key string, value *int) {
		if value != nil {
			values[name] = fileValue{key: key, text: strconv.Itoa(*value), list: nil}
		}
	}

	text(flags.Proxy, "proxy", f.Proxy)
	text(flags.Source, "source", f.Source)
	boolean(flags.DNSBindSource, "dnsBindSource", f.DNSBindSource)
	text(flags.UserAgent, "ua", f.UserAgent)
	text(flags.Location, "location", f.Location)
	text(flags.City, "city", f.City)
	text(flags.Search, "search", f.Search)
	number(flags.DSCP, "dscp", f.DSCP)
	boolean(flags.MaskISP, "maskISP", f.MaskISP)
	boolean(flags.Debug, "debug", f.Debug)
	text(flags.BaseURL, "baseURL", f.BaseURL)
	text(flags.CustomURL, "customURL", f.CustomURL)
	boolean(flags.SavingMode, "savingMode", f.SavingMode)
	boolean(flags.JSON, "json", f.JSON)
	boolean(flags.JSONL, "jsonl", f.JSONL)
	boolean(flags.Unix, "unix", f.Unix)
	boolean(flags.Multi, "multi", f.Multi)
	number(flags.Thread, "thread", f.Thread)
	boolean(flags.NoDownload, "noDownload", f.NoDownload)
	boolean(flags.NoUpload, "noUpload", f.NoUpload)
	text(flags.PingMode, "pingMode", f.PingMode)
	text(flags.Protocol, "protocol", f.Protocol)
	number(flags.DownloadSize, "dlSize", f.DownloadSize)
	number(flags.UploadSize, "ulSize", f.UploadSize)
	text(flags.Unit, "unit", f.Unit)

	if f.FilterCC != nil {
		values[flags.FilterCC] = fileValue{key: "filterCC", text: "", list: *f.FilterCC}
	}

	if f.Server != nil {
		ids := make([]string, 0, len(*f.Server))
		for _, id := range *f.Server {
			ids = append(ids, strconv.Itoa(id))
		}

		values[flags.Server] = fileValue{key: "server", text: "", list: ids}
	}

	return values
}

// readFile reads a settings file of at most maxFileSize bytes.
//
// Parameters:
//   - path: the file to read.
//
// Returns:
//   - []byte: the content.
//   - error: the open or read error, or errFileTooLarge.
func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open settings file: %w", err)
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("read settings file: %w", err)
	}

	if len(data) > maxFileSize {
		return nil, fmt.Errorf("%w: more than %d bytes", errFileTooLarge, maxFileSize)
	}

	return data, nil
}

// parseDocument parses settings content and checks its shape.
//
// Parameters:
//   - data: the YAML content.
//
// Returns:
//   - *yaml.Node: the document node, or nil when there is no document.
//   - error: an error wrapping ErrInvalidFile.
func parseDocument(data []byte) (*yaml.Node, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFile, errNotUTF8)
	}

	loader, err := yaml.NewLoader(bytes.NewReader(data), yaml.WithUniqueKeys(true))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	var doc yaml.Node

	err = loader.Load(&doc)
	if errors.Is(err, io.EOF) {
		return nil, nil //nolint:nilnil // Content without a document is valid and empty.
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	err = rejectMoreDocuments(loader)
	if err != nil {
		return nil, err
	}

	root := doc.Content[0]
	if isNull(root) {
		return &doc, nil
	}

	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFile, errNotMapping)
	}

	err = checkNodes(root)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	return &doc, nil
}

// checkNodes rejects anchors, aliases, and merge keys anywhere below node, which tie values together in ways the
// strict decoding cannot check.
//
// Parameters:
//   - node: the node to check.
//
// Returns:
//   - error: errUnsupportedYAML naming the line.
func checkNodes(node *yaml.Node) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode || node.Tag == "!!merge" {
		return fmt.Errorf("%w at line %d", errUnsupportedYAML, node.Line)
	}

	for _, child := range node.Content {
		err := checkNodes(child)
		if err != nil {
			return err
		}
	}

	return nil
}

// isNull reports whether a node is the null scalar, such as "~", with no anchor.
//
// Parameters:
//   - node: a parsed node.
//
// Returns:
//   - bool: true for an unanchored null.
func isNull(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null" && node.Anchor == ""
}

// rejectMoreDocuments reads the rest of a stream and fails on any document with content. Empty documents, such as
// one left by a trailing "---", pass.
//
// Parameters:
//   - loader: the loader after the first document.
//
// Returns:
//   - error: an error wrapping ErrInvalidFile.
func rejectMoreDocuments(loader *yaml.Loader) error {
	for {
		var extra any

		err := loader.Load(&extra)

		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return fmt.Errorf("%w: %w", ErrInvalidFile, err)
		case extra != nil:
			return fmt.Errorf("%w: %w", ErrInvalidFile, errMultipleDocuments)
		default:
		}
	}
}
