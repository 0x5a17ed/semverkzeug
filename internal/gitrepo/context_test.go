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
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func must[T any](t *testing.T) func(v T, err error) T {
	return func(v T, err error) T {
		require.NoError(t, err)
		return v
	}
}

func TestContext_String(t *testing.T) {
	type args struct {
		repo *git.Repository

		equalFn func(t *testing.T, b string)
	}
	tests := []struct {
		name string
		args args
	}{
		{"zero value", args{
			repo: nil,
			equalFn: func(t *testing.T, b string) {
				assert.Equal(t, "gitrepo.Context{repo=<nil>}", b)
			},
		}},
		{"memory", args{
			repo: func() *git.Repository {
				return must[*git.Repository](t)(
					git.Init(memory.NewStorage(), nil),
				)
			}(),
			equalFn: func(t *testing.T, b string) {
				assert.Equal(t, "gitrepo.Context{repo=<memory>}", b)
			},
		}},
		{"filesystem", args{
			repo: func() *git.Repository {
				return must[*git.Repository](t)(
					git.PlainInit(t.TempDir(), false),
				)
			}(),
			equalFn: func(t *testing.T, b string) {
				assert.Regexp(t, `gitrepo.Context{repo=<filesystem:path=.*}`, b)
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cx, err := NewContextFromRepo(tt.args.repo)
			require.NoError(t, err)

			tt.args.equalFn(t, cx.String())
		})
	}
}
