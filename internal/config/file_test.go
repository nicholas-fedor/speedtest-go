// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/flags"
)

// TestDecodeFile checks that every setting decodes, and that values() keys each one by its flag name with its key in
// the file.
func TestDecodeFile(t *testing.T) {
	t.Parallel()

	file, err := DecodeFile([]byte(`
proxy: http://proxy:8080
source: 192.0.2.10
dnsBindSource: true
ua: test-agent
location: "35,139"
city: tokyo
search: fiber
filterCC: [gb, ie]
dscp: 46
maskISP: true
debug: false
baseURL: http://127.0.0.1:8080
server: [1001, 1002]
customURL: http://speed.example
savingMode: true
json: true
jsonl: false
unix: false
multi: true
thread: 4
noDownload: true
noUpload: false
pingMode: tcp
protocol: tcp
dlSize: 2000
ulSize: 300
unit: binary-bits
`))
	require.NoError(t, err)

	values := file.values()
	assert.Len(t, values, 27, "every setting except the settings file itself")

	assert.Equal(t, fileValue{key: "pingMode", text: "tcp", list: nil}, values[flags.PingMode])
	assert.Equal(t, fileValue{key: "thread", text: "4", list: nil}, values[flags.Thread])
	assert.Equal(
		t,
		fileValue{key: "dnsBindSource", text: "true", list: nil},
		values[flags.DNSBindSource],
	)
	assert.Equal(
		t,
		fileValue{key: "server", text: "", list: []string{"1001", "1002"}},
		values[flags.Server],
	)
	assert.Equal(
		t,
		fileValue{key: "filterCC", text: "", list: []string{"gb", "ie"}},
		values[flags.FilterCC],
	)
}

// TestDecodeFile_Empty checks that content without settings holds none.
func TestDecodeFile_Empty(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"empty":          "",
		"comments":       "# no settings yet\n",
		"null":           "~\n",
		"trailing empty": "json: true\n---\n",
	} {
		file, err := DecodeFile([]byte(content))
		require.NoError(t, err, name)

		if name == "trailing empty" {
			assert.Len(t, file.values(), 1, name)

			continue
		}

		assert.Empty(t, file.values(), name)
	}
}

// TestDecodeFile_Rejects checks the strict decoding: unknown keys, repeated keys, wrong types, more than one
// document, anchors and aliases, a document that is not a mapping, and content that is not UTF-8 are all errors.
func TestDecodeFile_Rejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		content string
	}{
		{name: "unknown key", content: "jsn: true\n"},
		{name: "flag spelling", content: "ping-mode: tcp\n"},
		{name: "repeated key", content: "json: true\njson: false\n"},
		{name: "wrong type", content: "thread: four\n"},
		{name: "list for a single value", content: "city: [tokyo]\n"},
		{
			name:    "two documents",
			content: "json: true\n---\nunix: true\n",
			wantErr: errMultipleDocuments,
		},
		{
			name:    "anchor",
			content: "city: &home tokyo\nsearch: *home\n",
			wantErr: errUnsupportedYAML,
		},
		{name: "not a mapping", content: "- json\n", wantErr: errNotMapping},
		{name: "not UTF-8", content: "city: \xff\n", wantErr: errNotUTF8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := DecodeFile([]byte(tt.content))
			require.ErrorIs(t, err, ErrInvalidFile)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

// TestLoadFile checks reading a settings file: one that must exist, one that may be missing, and one too large.
func TestLoadFile(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "config.yaml")

	_, err := LoadFile(missing)
	require.ErrorIs(t, err, ErrInvalidFile)
	require.ErrorIs(t, err, fs.ErrNotExist)

	file, err := LoadOptionalFile(missing)
	require.NoError(t, err)
	assert.Empty(t, file.values())

	large := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(large, bytes.Repeat([]byte("# padding\n"), maxFileSize), 0o600))

	_, err = LoadOptionalFile(large)
	require.ErrorIs(t, err, errFileTooLarge)

	invalid := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(invalid, []byte("jsn: true\n"), 0o600))

	_, err = LoadFile(invalid)
	require.ErrorIs(t, err, ErrInvalidFile)
	assert.Contains(t, err.Error(), invalid, "the error names the file")
}
