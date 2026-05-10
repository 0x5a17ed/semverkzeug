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
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

func maxEntryMTime(entries []gitrepo.DirtyEntry) time.Time {
	var t time.Time
	for _, e := range entries {
		if e.ModTime().After(t) {
			t = e.ModTime()
		}
	}
	return t
}

func collectErr[T any](s iter.Seq[T], doneFn func() error) ([]T, error) {
	out := slices.Collect(s)
	return out, doneFn()
}

func TestFindDirtyEntries(t *testing.T) {
	t.Run("clean repo returns no entries", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)

		entries, err := collectErr(gitrepo.IterDirtyEntries(cx))
		require.NoError(t, err)

		assert.Empty(t, entries)
	})

	t.Run("considers untracked files", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)

		gitfixture.WriteRepoFile(t, cx, "/baa", "baz")

		inf, err := gitfixture.Filesystem(t, cx).Stat("/baa")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		entries, err := collectErr(gitrepo.IterDirtyEntries(cx))
		require.NoError(t, err)

		assert.WithinDuration(t, after, maxEntryMTime(entries), 100*time.Millisecond)
	})

	t.Run("modified file mtime", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)

		// Modify the file to ensure the repo reports a dirty state.
		gitfixture.WriteRepoFile(t, cx, "/foo", "baz")

		inf, err := gitfixture.Filesystem(t, cx).Stat("/foo")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		entries, err := collectErr(gitrepo.IterDirtyEntries(cx))
		require.NoError(t, err)

		assert.WithinDuration(t, after, maxEntryMTime(entries), 100*time.Millisecond)
	})

	t.Run("deleted file reports parent directory mtime", func(t *testing.T) {
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)

		dirInfoBefore, err := gitfixture.Filesystem(t, cx).Stat("/")
		require.NoError(t, err)
		before := dirInfoBefore.ModTime().UTC()

		err = gitfixture.Filesystem(t, cx).Remove("/foo")
		require.NoError(t, err)

		dirInfoAfter, err := gitfixture.Filesystem(t, cx).Stat("/")
		require.NoError(t, err)
		after := dirInfoAfter.ModTime().UTC()

		// Optional sanity check: never goes backwards.
		assert.False(t, after.Before(before), "mtime should not go backwards")

		entries, err := collectErr(gitrepo.IterDirtyEntries(cx))
		require.NoError(t, err)

		assert.WithinDuration(t, after, maxEntryMTime(entries), 100*time.Millisecond)
	})

	t.Run("deleted directory reports parent directory mtime", func(t *testing.T) {
		cx := gitfixture.RepoEmpty(t)

		gitfixture.CommitFile(t, cx, "foo/baa/baz", "bar")

		err := util.RemoveAll(gitfixture.Filesystem(t, cx), "/foo")
		require.NoError(t, err)

		inf, err := gitfixture.Filesystem(t, cx).Stat("/")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		entries, err := collectErr(gitrepo.IterDirtyEntries(cx))
		require.NoError(t, err)

		assert.WithinDuration(t, after, maxEntryMTime(entries), 100*time.Millisecond)
	})
}
