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

package gitrepo_test

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/testhelper"
)

func flattenNames(tm gitrepo.VersionTagMap, h plumbing.Hash) []string {
	names := make([]string, 0, len(tm))
	for _, tags := range tm {
		for _, tag := range tags {
			if h != plumbing.ZeroHash {
				if tag.CommitHash != h {
					continue
				}
			}
			names = append(names, tag.VersionSpec.String())
		}
	}
	return names
}

func TestGetTagMap(t *testing.T) {
	t.Run("no-tags", func(t *testing.T) {
		// Arrange: repo with one commit, no tags.
		repo := testhelper.RepoEmpty(t)
		testhelper.CommitFile(t, repo, "readme.txt", "hello")

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo.Repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, tm)
	})

	t.Run("lightweight-tag", func(t *testing.T) {
		// Arrange: repo with one commit and a lightweight tag.
		repo := testhelper.RepoEmpty(t)
		h := testhelper.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repo.CreateTag("v1.0.0", h, nil)
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo.Repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v1.0.0"}, flattenNames(tm, h))
	})

	t.Run("annotated-tag", func(t *testing.T) {
		// Arrange: repo with one commit and an annotated tag.
		repo := testhelper.RepoEmpty(t)
		h := testhelper.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repo.CreateTag("v2.0.0", h, &git.CreateTagOptions{
			Tagger:  testhelper.TestSig,
			Message: "release v2.0.0",
		})
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo.Repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v2.0.0"}, flattenNames(tm, h))
	})

	t.Run("multiple-tags-same-commit", func(t *testing.T) {
		// Arrange: two tags pointing at the same commit.
		repo := testhelper.RepoEmpty(t)
		h := testhelper.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repo.CreateTag("v1.0.0", h, nil)
		require.NoError(t, err)
		_, err = repo.Repo.CreateTag("mod/v1.0.0", h, nil)
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo.Repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, tm[h], 2)
		assert.ElementsMatch(t, []string{"v1.0.0", "mod/v1.0.0"}, flattenNames(tm, h))
	})

	t.Run("tags-on-different-commits", func(t *testing.T) {
		// Arrange: two commits, each with its own tag.
		repo := testhelper.RepoEmpty(t)
		h1 := testhelper.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repo.CreateTag("v1.0.0", h1, nil)
		require.NoError(t, err)

		h2 := testhelper.CommitFile(t, repo, "b.txt", "b")
		_, err = repo.Repo.CreateTag("v2.0.0", h2, &git.CreateTagOptions{
			Tagger:  testhelper.TestSig,
			Message: "release v2.0.0",
		})
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo.Repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v1.0.0"}, flattenNames(tm, h1))
		assert.Equal(t, []string{"v2.0.0"}, flattenNames(tm, h2))
	})
}
