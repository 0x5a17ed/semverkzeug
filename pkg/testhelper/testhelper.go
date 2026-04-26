package testhelper

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
)

var TestSig = &object.Signature{
	Name:  "test",
	Email: "test@example.test",
	When:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

type Repo struct {
	Repo       *git.Repository
	WorkTree   *git.Worktree
	Filesystem billy.Filesystem
}

func RepoEmpty(t *testing.T) *Repo {
	t.Helper()

	repo, err := git.PlainInitWithOptions(t.TempDir(), &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.Main,
		},
		ObjectFormat: config.SHA1,
	})
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	return &Repo{
		Repo:       repo,
		WorkTree:   wt,
		Filesystem: wt.Filesystem,
	}
}

func WriteFile(t *testing.T, repo *Repo, name string, data []byte) {
	t.Helper()

	err := repo.Filesystem.MkdirAll(filepath.Dir(name), 0755)
	require.NoError(t, err, "failed to create directory %s", filepath.Dir(name))

	f, err := repo.Filesystem.Create(name)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	_, err = f.Write(data)
	require.NoError(t, err)
}

func CommitFile(t *testing.T, repo *Repo, name, content string) plumbing.Hash {
	t.Helper()

	WriteFile(t, repo, name, []byte(content))

	require.NoError(t, repo.WorkTree.AddWithOptions(&git.AddOptions{Path: name}))

	h, err := repo.WorkTree.Commit("commit "+name, &git.CommitOptions{
		Author:    TestSig,
		Committer: TestSig,
	})
	require.NoError(t, err)

	return h
}

func RepoNoCommitsDirty(t *testing.T) *Repo {
	scope := RepoEmpty(t)

	WriteFile(t, scope, "foo", []byte("baa"))

	return scope
}

func RepoOneCommitClean(t *testing.T) *Repo {
	scope := RepoNoCommitsDirty(t)

	err := scope.WorkTree.AddWithOptions(&git.AddOptions{Path: "foo"})
	require.NoError(t, err)

	_, err = scope.WorkTree.Commit("asd", &git.CommitOptions{
		Author:    TestSig,
		Committer: TestSig,
	})
	require.NoError(t, err)

	return scope
}
