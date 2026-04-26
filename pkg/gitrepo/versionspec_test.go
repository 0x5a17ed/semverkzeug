/*
 * Copyright(C) 2022 individual contributors
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersionTag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		scope   string
		prefix  string
		version string
		wantErr bool
	}{
		{
			name:    "plain-semver",
			input:   "1.2.3",
			version: "1.2.3",
		},
		{
			name:    "v-prefix",
			input:   "v1.0.0",
			prefix:  "v",
			version: "1.0.0",
		},
		{
			name:    "scoped-tag",
			input:   "mod/v2.3.4",
			scope:   "mod",
			prefix:  "v",
			version: "2.3.4",
		},
		{
			name:    "nested-scope",
			input:   "pkg/sub/v0.1.0",
			scope:   "pkg/sub",
			prefix:  "v",
			version: "0.1.0",
		},
		{
			name:    "prerelease",
			input:   "v1.0.0-alpha.1",
			prefix:  "v",
			version: "1.0.0-alpha.1",
		},
		{
			name:    "build-metadata",
			input:   "v1.0.0+build.42",
			prefix:  "v",
			version: "1.0.0+build.42",
		},
		{
			name:    "prerelease-and-build",
			input:   "v1.0.0-rc.1+sha.abc",
			prefix:  "v",
			version: "1.0.0-rc.1+sha.abc",
		},
		{
			name:    "empty-string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "not-a-version",
			input:   "hello",
			wantErr: true,
		},
		{
			name:    "incomplete-version",
			input:   "v1.2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got, err := ParseVersionSpec(tt.input)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.input, got.String())
			assert.Equal(t, tt.scope, got.Scope.String())
			assert.Equal(t, tt.prefix, got.Prefix)
			assert.Equal(t, tt.version, got.Version.String())
		})
	}
}
