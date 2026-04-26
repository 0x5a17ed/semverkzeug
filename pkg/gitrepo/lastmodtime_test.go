package gitrepo

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/testhelper"
)

func TestLastModificationTime(t *testing.T) {
	t.Run("clean repo returns error", func(t *testing.T) {
		scope := testhelper.RepoOneCommitClean(t)

		st, err := scope.WorkTree.Status()
		require.NoError(t, err)

		mtime, err := FindWorktreeMTime(scope.WorkTree, st)
		require.ErrorIs(t, err, ErrNotDirty)

		assert.Nil(t, mtime)
	})

	t.Run("consider untracked files", func(t *testing.T) {
		scope := testhelper.RepoOneCommitClean(t)

		testhelper.WriteFile(t, scope, "/baa", []byte("baz"))

		inf, err := scope.Filesystem.Stat("/baa")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		st, err := scope.WorkTree.Status()
		require.NoError(t, err)

		// Assert that the mtime of the untracked file is reported.
		mtime, err := FindWorktreeMTime(scope.WorkTree, st)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})

	t.Run("modified file mtime", func(t *testing.T) {
		scope := testhelper.RepoOneCommitClean(t)

		// Modify the file to ensure the repo reports a dirty state.
		testhelper.WriteFile(t, scope, "/foo", []byte("baz"))

		inf, err := scope.Filesystem.Stat("/foo")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		st, err := scope.WorkTree.Status()
		require.NoError(t, err)

		mtime, err := FindWorktreeMTime(scope.WorkTree, st)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})

	t.Run("deleted file reports parent directory mtime", func(t *testing.T) {
		scope := testhelper.RepoOneCommitClean(t)

		dirInfoBefore, err := scope.Filesystem.Stat("/")
		require.NoError(t, err)
		before := dirInfoBefore.ModTime().UTC()

		err = scope.Filesystem.Remove("/foo")
		require.NoError(t, err)

		dirInfoAfter, err := scope.Filesystem.Stat("/")
		require.NoError(t, err)
		after := dirInfoAfter.ModTime().UTC()

		// Optional sanity check: never goes backwards.
		assert.False(t, after.Before(before), "mtime should not go backwards")

		st, err := scope.WorkTree.Status()
		require.NoError(t, err)

		mtime, err := FindWorktreeMTime(scope.WorkTree, st)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})

	t.Run("deleted directory reports parent directory mtime", func(t *testing.T) {
		scope := testhelper.RepoEmpty(t)

		testhelper.CommitFile(t, scope, "foo/baa/baz", "bar")

		err := util.RemoveAll(scope.Filesystem, "/foo")
		require.NoError(t, err)

		inf, err := scope.Filesystem.Stat("/")
		require.NoError(t, err)
		after := inf.ModTime().UTC()

		st, err := scope.WorkTree.Status()
		require.NoError(t, err)

		mtime, err := FindWorktreeMTime(scope.WorkTree, st)
		require.NoError(t, err)

		assert.WithinDuration(t, after, *mtime, 100*time.Millisecond)
	})
}
