/*
 * Copyright(C) 2026 the semverkzeug developers
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * <https://www.apache.org/licenses/LICENSE-2.0>
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied.  See the License for the specific
 * language governing permissions and limitations under the License.
 */

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMessageSource(t *testing.T) {
	tt := []struct {
		name    string
		raw     string
		want    MessageSource
		wantErr bool
	}{
		{
			name: "dash selects standard input",
			raw:  "-",
			want: MessageSource{variant: MessageFromStdin{}},
		},
		{
			name: "path selects file",
			raw:  "notes.md",
			want: MessageSource{variant: MessageFromFile{path: "notes.md"}},
		},
		{
			name: "path with surrounding spaces is kept verbatim",
			raw:  " notes.md ",
			want: MessageSource{variant: MessageFromFile{path: " notes.md "}},
		},
		{
			name:    "empty path is rejected",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMessageSource(tc.raw)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// writeMessageFile writes content to a file in a fresh temporary
// directory and returns the path of the file.
func writeMessageFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "notes.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestMessageSourceRead(t *testing.T) {
	const notes = "# Release\n\nNotes.\n"

	tt := []struct {
		name    string
		source  MessageSource
		stdin   string
		want    string
		wantErr error
	}{
		{
			name:   "zero value selects generated message",
			source: MessageSource{},
			stdin:  notes,
			want:   "",
		},
		{
			name:   "standard input is read verbatim",
			source: MessageSource{variant: MessageFromStdin{}},
			stdin:  notes,
			want:   notes,
		},
		{
			name:    "blank standard input is rejected",
			source:  MessageSource{variant: MessageFromStdin{}},
			stdin:   " \n\t\n",
			wantErr: errEmptyMessage,
		},
		{
			name:   "file is read verbatim",
			source: MessageSource{variant: MessageFromFile{path: writeMessageFile(t, notes)}},
			want:   notes,
		},
		{
			name:    "blank file is rejected",
			source:  MessageSource{variant: MessageFromFile{path: writeMessageFile(t, " \n\t\n")}},
			wantErr: errEmptyMessage,
		},
		{
			name:    "missing file is rejected",
			source:  MessageSource{variant: MessageFromFile{path: filepath.Join(t.TempDir(), "missing.md")}},
			wantErr: os.ErrNotExist,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.source.Read(strings.NewReader(tc.stdin))
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBumpMessageFileFlag(t *testing.T) {
	tt := []struct {
		name    string
		args    []string
		want    MessageSource
		wantErr bool
	}{
		{
			name: "absent flag selects generated message",
			args: []string{"bump", "patch"},
			want: MessageSource{},
		},
		{
			name: "dash selects standard input",
			args: []string{"bump", "patch", "-F", "-"},
			want: MessageSource{variant: MessageFromStdin{}},
		},
		{
			name: "path selects file",
			args: []string{"bump", "patch", "--message-file", "notes.md"},
			want: MessageSource{variant: MessageFromFile{path: "notes.md"}},
		},
		{
			name:    "empty path is rejected",
			args:    []string{"bump", "patch", "--message-file="},
			wantErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var grammar cli
			parser, err := kong.New(&grammar)
			require.NoError(t, err)

			_, err = parser.Parse(tc.args)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, grammar.Bump.MessageFile)
		})
	}
}
