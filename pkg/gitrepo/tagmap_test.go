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
	"github.com/0x5a17ed/semverkzeug/pkg/internal/gitfixture"
)

func filterNamesInVersionTagMap(tm gitrepo.VersionTagMap, h plumbing.Hash) []string {
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
		repo := gitfixture.RepoEmpty(t)
		gitfixture.CommitFile(t, repo, "readme.txt", "hello")

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, tm)
	})

	t.Run("lightweight-tag", func(t *testing.T) {
		// Arrange: repo with one commit and a lightweight tag.
		repo := gitfixture.RepoEmpty(t)
		c := gitfixture.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repository().CreateTag("v1.0.0", c.Hash, nil)
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v1.0.0"}, filterNamesInVersionTagMap(tm, c.Hash))
	})

	t.Run("annotated-tag", func(t *testing.T) {
		// Arrange: repo with one commit and an annotated tag.
		repo := gitfixture.RepoEmpty(t)
		c := gitfixture.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repository().CreateTag("v2.0.0", c.Hash, &git.CreateTagOptions{
			Tagger:  gitfixture.TestSig,
			Message: "release v2.0.0",
		})
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v2.0.0"}, filterNamesInVersionTagMap(tm, c.Hash))
	})

	t.Run("multiple-tags-same-commit", func(t *testing.T) {
		// Arrange: two tags pointing at the same commit.
		repo := gitfixture.RepoEmpty(t)
		c := gitfixture.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repository().CreateTag("v1.0.0", c.Hash, nil)
		require.NoError(t, err)
		_, err = repo.Repository().CreateTag("mod/v1.0.0", c.Hash, nil)
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, tm[c.Hash], 2)
		assert.ElementsMatch(t, []string{"v1.0.0", "mod/v1.0.0"}, filterNamesInVersionTagMap(tm, c.Hash))
	})

	t.Run("tags-on-different-commits", func(t *testing.T) {
		// Arrange: two commits, each with its own tag.
		repo := gitfixture.RepoEmpty(t)
		c1 := gitfixture.CommitFile(t, repo, "a.txt", "a")
		_, err := repo.Repository().CreateTag("v1.0.0", c1.Hash, nil)
		require.NoError(t, err)

		c2 := gitfixture.CommitFile(t, repo, "b.txt", "b")
		_, err = repo.Repository().CreateTag("v2.0.0", c2.Hash, &git.CreateTagOptions{
			Tagger:  gitfixture.TestSig,
			Message: "release v2.0.0",
		})
		require.NoError(t, err)

		// Act
		tm, err := gitrepo.NewVersionTagMapFromRepo(repo, nil)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v1.0.0"}, filterNamesInVersionTagMap(tm, c1.Hash))
		assert.Equal(t, []string{"v2.0.0"}, filterNamesInVersionTagMap(tm, c2.Hash))
	})
}
