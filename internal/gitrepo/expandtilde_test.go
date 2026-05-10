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

package gitrepo

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  func(home string) string
	}{
		{
			name:  "bare tilde expands to home",
			input: "~",
			want: func(home string) string {
				return home
			},
		},
		{
			name:  "tilde slash expands under home",
			input: "~/git/ignore",
			want: func(home string) string {
				return filepath.Join(home, "git", "ignore")
			},
		},
		{
			name:  "relative path without leading tilde is unchanged",
			input: "git/ignore",
			want: func(_ string) string {
				return "git/ignore"
			},
		},
		{
			name:  "embedded tilde is unchanged",
			input: "git/~/ignore",
			want: func(_ string) string {
				return "git/~/ignore"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			home := t.TempDir()
			t.Setenv("HOME", home)
			want := tt.want(home)

			// Act
			got, err := expandTilde(tt.input)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestExpandTildeErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		arrange func(t *testing.T)
		wantErr string
	}{
		{
			name:  "user tilde is unsupported",
			input: "~alice",
			arrange: func(t *testing.T) {
				t.Helper()
				t.Setenv("HOME", t.TempDir())
			},
			wantErr: `unsupported tilde expansion "~alice"`,
		},
		{
			name:  "user tilde with path is unsupported",
			input: "~alice/git/ignore",
			arrange: func(t *testing.T) {
				t.Helper()
				t.Setenv("HOME", t.TempDir())
			},
			wantErr: `unsupported tilde expansion "~alice/git/ignore"`,
		},
		{
			name:  "home directory cannot be resolved",
			input: "~",
			arrange: func(t *testing.T) {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("HOME does not control os.UserHomeDir on Windows")
				}
				t.Setenv("HOME", "")
			},
			wantErr: "resolve home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tt.arrange(t)

			// Act
			got, err := expandTilde(tt.input)

			// Assert
			require.Error(t, err)
			assert.Empty(t, got)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
