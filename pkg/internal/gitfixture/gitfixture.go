package gitfixture

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

var TestSig = &object.Signature{
	Name:  "test",
	Email: "test@example.test",
	When:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

func Worktree(t *testing.T, cx *gitrepo.Context) *git.Worktree {
	t.Helper()

	wt, err := cx.LoadWorktree()
	require.NoError(t, err)

	return wt
}

func Filesystem(t *testing.T, cx *gitrepo.Context) billy.Filesystem {
	t.Helper()

	wt := Worktree(t, cx)

	return wt.Filesystem
}

func WriteFile(t *testing.T, gCx *gitrepo.Context, name, text string) {
	t.Helper()

	wt := Worktree(t, gCx)

	err := wt.Filesystem.MkdirAll(filepath.Dir(name), 0755)
	require.NoError(t, err, "failed to create directory %s", filepath.Dir(name))

	f, err := wt.Filesystem.Create(name)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	_, err = f.Write([]byte(text))
	require.NoError(t, err)
}

func CommitFile(t *testing.T, gCx *gitrepo.Context, name, content string) plumbing.Hash {
	t.Helper()

	wt := Worktree(t, gCx)

	WriteFile(t, gCx, name, content)

	require.NoError(t, wt.AddWithOptions(&git.AddOptions{Path: name}))

	h, err := wt.Commit("commit "+name, &git.CommitOptions{
		Author:    TestSig,
		Committer: TestSig,
	})
	require.NoError(t, err)

	return h
}

func Checkout(t *testing.T, gCx *gitrepo.Context, name string, create bool) {
	t.Helper()

	require.NoError(t, Worktree(t, gCx).Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
		Create: create,
	}))
}

func Head(t *testing.T, gCx *gitrepo.Context) *plumbing.Reference {
	t.Helper()

	ref, err := gCx.Repository().Head()
	require.NoError(t, err)

	return ref
}

func CreateTag(t *testing.T, gCx *gitrepo.Context, name string) plumbing.Hash {
	t.Helper()

	ref, err := gCx.Repository().CreateTag(name, Head(t, gCx).Hash(), &git.CreateTagOptions{
		Tagger:  TestSig,
		Message: "tagged commit",
	})
	require.NoError(t, err)

	return ref.Hash()
}

func RepoEmpty(t *testing.T) *gitrepo.Context {
	t.Helper()

	repo, err := git.PlainInitWithOptions(t.TempDir(), &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.Main,
		},
		ObjectFormat: config.SHA1,
	})
	require.NoError(t, err)

	cx, err := gitrepo.NewContextFromRepo(repo)
	require.NoError(t, err)

	return cx
}

func RepoWithNoCommitsNoTagsDirty(t *testing.T) *gitrepo.Context {
	scope := RepoEmpty(t)

	WriteFile(t, scope, "foo", "baa")

	return scope
}

func RepoWithOneCommitNoTagsClean(t *testing.T) *gitrepo.Context {
	cx := RepoEmpty(t)

	CommitFile(t, cx, "foo", "baa")

	return cx
}

func RepoWithOneCommitNoTagsFileDeleted(t *testing.T) *gitrepo.Context {
	cx := RepoWithOneCommitNoTagsClean(t)

	require.NoError(t, Filesystem(t, cx).Remove("foo"))

	return cx
}

func RepoWithOneCommitNoTagsDirty(t *testing.T) *gitrepo.Context {
	cx := RepoWithOneCommitNoTagsClean(t)

	WriteFile(t, cx, "foo", "baz")

	return cx
}

func RepoWithOneCommitOneTagClean(t *testing.T) *gitrepo.Context {
	cx := RepoWithOneCommitNoTagsClean(t)

	CreateTag(t, cx, "v0.1.0")

	return cx
}

func RepoWithOneCommitOneTagDirty(t *testing.T) *gitrepo.Context {
	cx := RepoWithOneCommitOneTagClean(t)

	WriteFile(t, cx, "foo", "baz")

	return cx
}

func RepoWithTwoCommitsOneTagClean(t *testing.T) *gitrepo.Context {
	cx := RepoWithOneCommitOneTagClean(t)

	CommitFile(t, cx, "bar", "baa")

	return cx
}

func RepoWithTwoCommitsOneTagDirty(t *testing.T) *gitrepo.Context {
	cx := RepoWithTwoCommitsOneTagClean(t)

	WriteFile(t, cx, "bar", "baz")

	return cx
}
