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

package gitrepo_test

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/internal/gitfixture"
)

func TestLastModificationTime(t *testing.T) {
	t.Run("clean repo returns error", func(t *testing.T) {
		scope := gitfixture.RepoWithOneCommitNoTagsClean(t)

		mtime, err := gitrepo.FindWorktreeMTime(scope)
		require.ErrorIs(t, err, gitrepo.ErrWorktreeClean)

		assert.Nil(t, mtime)
	})

	t.Run("consider untracked files", func(t *testing.T) {
		scope := gitfixture.RepoWithOneCommitNoTagsClean(t)

		gitfixture.WriteFile(t, scope, "/baa", "baz")

		inf, err := gitfixture.Worktree(t, scope).Filesystem.Stat("/baa")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		// Assert that the mtime of the untracked file is reported.
		mtime, err := gitrepo.FindWorktreeMTime(scope)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})

	t.Run("modified file mtime", func(t *testing.T) {
		scope := gitfixture.RepoWithOneCommitNoTagsClean(t)

		// Modify the file to ensure the repo reports a dirty state.
		gitfixture.WriteFile(t, scope, "/foo", "baz")

		inf, err := gitfixture.Filesystem(t, scope).Stat("/foo")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		mtime, err := gitrepo.FindWorktreeMTime(scope)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})

	t.Run("deleted file reports parent directory mtime", func(t *testing.T) {
		scope := gitfixture.RepoWithOneCommitNoTagsClean(t)

		dirInfoBefore, err := gitfixture.Filesystem(t, scope).Stat("/")
		require.NoError(t, err)
		before := dirInfoBefore.ModTime().UTC()

		err = gitfixture.Filesystem(t, scope).Remove("/foo")
		require.NoError(t, err)

		dirInfoAfter, err := gitfixture.Filesystem(t, scope).Stat("/")
		require.NoError(t, err)
		after := dirInfoAfter.ModTime().UTC()

		// Optional sanity check: never goes backwards.
		assert.False(t, after.Before(before), "mtime should not go backwards")

		mtime, err := gitrepo.FindWorktreeMTime(scope)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})

	t.Run("deleted directory reports parent directory mtime", func(t *testing.T) {
		scope := gitfixture.RepoEmpty(t)

		gitfixture.CommitFile(t, scope, "foo/baa/baz", "bar")

		err := util.RemoveAll(gitfixture.Filesystem(t, scope), "/foo")
		require.NoError(t, err)

		inf, err := gitfixture.Filesystem(t, scope).Stat("/")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		mtime, err := gitrepo.FindWorktreeMTime(scope)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})
}
