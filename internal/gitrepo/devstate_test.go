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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

func statePath(t *testing.T, cx *gitrepo.Context) string {
	t.Helper()
	return filepath.Join(
		gitfixture.Filesystem(t, cx).Root(),
		".git", "semverkzeug", "state.json",
	)
}

func TestFindStableWorktreeMTime(t *testing.T) {
	t.Run("clean repo returns ErrWorktreeClean", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)

		mtime, err := gitrepo.FindStableWorktreeMTime(cx)
		require.ErrorIs(t, err, gitrepo.ErrWorktreeClean)
		assert.Nil(t, mtime)
	})

	t.Run("first call persists state file", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)
		gitfixture.WriteFile(t, cx, "foo", "baz")

		_, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)

		info, err := os.Stat(statePath(t, cx))
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	t.Run("stability: index mtime drift does not perturb output", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)
		gitfixture.WriteFile(t, cx, "foo", "baz")

		first, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)
		require.NotNil(t, first)

		// Simulate a stray `git status` rewriting .git/index with a
		// fresher mtime. The fingerprint must absorb this and the
		// emitted value must stay put.
		indexPath := filepath.Join(gitfixture.Filesystem(t, cx).Root(), ".git", "index")
		future := first.Add(1 * time.Hour)
		require.NoError(t, os.Chtimes(indexPath, future, future))

		second, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)
		require.NotNil(t, second)

		assert.True(t, second.Equal(*first),
			"expected stability across calls: first=%s second=%s", first, second)
	})

	t.Run("monotonicity: backwards candidate yields floor + tick", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)
		gitfixture.WriteFile(t, cx, "foo", "baz")

		first, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)
		require.NotNil(t, first)

		// Make a real change (different content, different mtime so the
		// fingerprint is invalidated) and force everything backwards.
		gitfixture.WriteFile(t, cx, "foo", "qux")
		wtRoot := gitfixture.Filesystem(t, cx).Root()
		past := first.Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(filepath.Join(wtRoot, "foo"), past, past))
		require.NoError(t, os.Chtimes(filepath.Join(wtRoot, ".git", "index"), past, past))

		second, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)
		require.NotNil(t, second)

		expected := first.Add(10 * time.Millisecond)
		assert.True(t, second.Equal(expected),
			"expected floor advance: first=%s expected=%s second=%s",
			first, expected, second)
	})

	t.Run("forward change advances normally", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)
		gitfixture.WriteFile(t, cx, "foo", "baz")

		first, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)
		require.NotNil(t, first)

		gitfixture.WriteFile(t, cx, "foo", "quux")
		wtRoot := gitfixture.Filesystem(t, cx).Root()
		future := first.Add(2 * time.Hour)
		require.NoError(t, os.Chtimes(filepath.Join(wtRoot, "foo"), future, future))

		second, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)
		require.NotNil(t, second)

		assert.False(t, second.Before(future),
			"expected forward advance: future=%s second=%s", future, second)
	})

	t.Run("repeated calls with no changes are idempotent", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)
		gitfixture.WriteFile(t, cx, "foo", "baz")

		first, err := gitrepo.FindStableWorktreeMTime(cx)
		require.NoError(t, err)

		for range 5 {
			next, err := gitrepo.FindStableWorktreeMTime(cx)
			require.NoError(t, err)
			require.NotNil(t, next)
			assert.True(t, next.Equal(*first),
				"expected idempotence: first=%s next=%s", first, next)
		}
	})
}
