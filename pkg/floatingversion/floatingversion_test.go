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

package floatingversion

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var signature = &object.Signature{
	Name:  "user",
	Email: "user@example.test",
	When:  time.Now(),
}

func writeFile(t *testing.T, fs billy.Filesystem, name string, data []byte) {
	f, err := fs.Create(name)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	_, err = f.Write(data)
	require.NoError(t, err)
}

func emptyFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	storer := memory.NewStorage()
	fs := memfs.New()

	repo, err := git.Init(storer, fs)
	require.NoError(t, err)
	return fs, repo
}

func emptyDirtyFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := emptyFixture(t)

	writeFile(t, fs, "foo", []byte("asdsdadsa"))

	return fs, repo
}

func oneCommitFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := emptyDirtyFixture(t)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	err = wt.AddWithOptions(&git.AddOptions{Path: "foo"})
	require.NoError(t, err)

	_, err = wt.Commit("asd", &git.CommitOptions{Author: signature, Committer: signature})
	require.NoError(t, err)

	return fs, repo
}

func oneCommitDirtyFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := oneCommitFixture(t)

	writeFile(t, fs, "foo", []byte("djgfshgjkdjfhkdf"))

	return fs, repo
}

func oneTaggedCommitRepositoryFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := oneCommitFixture(t)

	head, err := repo.Head()
	require.NoError(t, err)

	_, err = repo.CreateTag("v0.1.0", head.Hash(), &git.CreateTagOptions{
		Tagger: signature, Message: "version v0.1.0",
	})
	require.NoError(t, err)

	return fs, repo
}

func oneTaggedDirtyFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := oneTaggedCommitRepositoryFixture(t)

	writeFile(t, fs, "foo", []byte("djgfshgjkdjfhkdf"))

	return fs, repo
}

func oneTagOneCommitFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := oneTaggedDirtyFixture(t)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	err = wt.AddWithOptions(&git.AddOptions{Path: "foo"})
	require.NoError(t, err)

	_, err = wt.Commit("asd", &git.CommitOptions{Author: signature, Committer: signature})
	require.NoError(t, err)

	return fs, repo
}

func oneTagOneCommitDirtyFixture(t *testing.T) (billy.Filesystem, *git.Repository) {
	fs, repo := oneTagOneCommitFixture(t)

	writeFile(t, fs, "foo", []byte("ghksdfjghksjdfhd"))

	return fs, repo
}

type repositoryProvider func(t *testing.T) (billy.Filesystem, *git.Repository)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name    string
		repo    repositoryProvider
		wantVer string
		wantErr assert.ErrorAssertionFunc
	}{
		{"empty", emptyFixture, `v0\.0\.1-dev\.0`, assert.NoError},
		{"empty-dirty", emptyDirtyFixture, `v0\.0\.1-dev\.0\.\d{14}`, assert.NoError},
		{"one-commit-no-tag", oneCommitFixture, `v0\.0\.1-dev\.1`, assert.NoError},
		{"one-commit-no-tag-dirty", oneCommitDirtyFixture, `v0\.0\.1-dev\.1\.\d{14}`, assert.NoError},
		{"one-tag", oneTaggedCommitRepositoryFixture, `v0\.1\.0`, assert.NoError},
		{"one-tag-dirty", oneTaggedDirtyFixture, `v0\.1\.1-dev\.0\.\d{14}`, assert.NoError},
		{"one-tag-one-commit", oneTagOneCommitFixture, `v0\.1\.1-dev\.1`, assert.NoError},
		{"one-tag-one-commit-dirty", oneTagOneCommitDirtyFixture, `v0\.1\.1-dev\.1\.\d{14}`, assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, repo := tt.repo(t)

			head, err := repo.Head()
			if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
				require.NoError(t, err)
			}

			gotVs, err := Get(repo, head, false)
			if !tt.wantErr(t, err, fmt.Sprintf("Get(%v)", tt.repo)) {
				return
			}

			assert.Regexp(t, tt.wantVer, gotVs.String())
		})
	}
}
