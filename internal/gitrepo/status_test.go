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
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

func requireStatus(t *testing.T, st git.Status, path string) *git.FileStatus {
	t.Helper()

	fs, ok := st[path]
	require.True(t, ok, "expected status entry for %#q; status:\n%s", path, st.String())
	return fs
}

func TestStatus_HonorsDefaultXDGIgnore(t *testing.T) {
	home := gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	// Arrange: ignore rule in default XDG config.
	gitfixture.WriteFile(t, home(".config", "git", "ignore"), "local/\n")

	cx := gitfixture.RepoEmpty(t)

	// Act: Commit a tracked file, write ignored file, build status.
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")

	gitfixture.WriteRepoFile(t, cx, "local/file", "ignored\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)

	// Assert: status should not contain ignored file.
	assert.NotContains(t, st, "local/file")
	assert.True(t, st.IsClean(), "status:\n%s", st.String())
}

func TestStatus_CoreExcludesFileReplacesDefaultXDGIgnore(t *testing.T) {
	home := gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	excludesFile := home("custom-ignore")
	gitfixture.WriteFile(t, excludesFile, "custom/\n")
	gitfixture.WriteFile(t, home(".config", "git", "config"),
		"[core]\n\texcludesFile = "+excludesFile+"\n")
	gitfixture.WriteFile(t, home(".config", "git", "ignore"),
		"xdgonly/\n")

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")
	gitfixture.WriteRepoFile(t, cx, "custom/file", "ignored\n")
	gitfixture.WriteRepoFile(t, cx, "xdgonly/file", "visible\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.NotContains(t, st, "custom/file")
	assert.Equal(t, git.Untracked, requireStatus(t, st, "xdgonly/file").Worktree)
}

func TestStatus_IncludeIfGitdirExpandsHome(t *testing.T) {
	home := gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	gitfixture.WriteFile(t, home("gitignore"), "local/\n")
	gitfixture.WriteFile(t, home("included-config"),
		"[core]\n\texcludesFile = ~/gitignore\n")
	gitfixture.WriteFile(t, home(".gitconfig"),
		"[includeIf \"gitdir:~/git/\"]\n\tpath = ~/included-config\n")

	repoPath := home("git", "repo")
	repo, err := git.PlainInitWithOptions(repoPath, &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.Main,
		},
		ObjectFormat: config.SHA1,
	})
	require.NoError(t, err)

	cx, err := gitrepo.NewContextFromRepo(repo)
	require.NoError(t, err)

	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")
	gitfixture.WriteRepoFile(t, cx, "local/file", "ignored\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.NotContains(t, st, "local/file")
	assert.True(t, st.IsClean(), "status:\n%s", st.String())
}

func TestStatus_GitignoreOverridesGlobalExclude(t *testing.T) {
	home := gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)
	gitfixture.WriteFile(t, home(".config", "git", "ignore"), "local/\n")

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")
	gitfixture.CommitFile(t, cx, ".gitignore", "!local/\n")
	gitfixture.WriteRepoFile(t, cx, "local/file", "visible\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.Equal(t, git.Untracked, requireStatus(t, st, "local/file").Worktree)
}

func TestStatus_DoesNotReincludeInsideIgnoredParent(t *testing.T) {
	home := gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)
	gitfixture.WriteFile(t, home(".config", "git", "ignore"), "local/\n")

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")
	gitfixture.CommitFile(t, cx, ".gitignore", "!local/file\n")
	gitfixture.WriteRepoFile(t, cx, "local/file", "ignored\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.NotContains(t, st, "local/file")
	assert.True(t, st.IsClean(), "status:\n%s", st.String())
}

func TestStatus_DoesNotUseGitignoreInsideIgnoredDirectoryToReinclude(t *testing.T) {
	_ = gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")
	gitfixture.CommitFile(t, cx, ".gitignore", "ignored/\n")
	gitfixture.WriteRepoFile(t, cx, "ignored/.gitignore", "!file\n")
	gitfixture.WriteRepoFile(t, cx, "ignored/file", "ignored\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.NotContains(t, st, "ignored/file")
	assert.True(t, st.IsClean(), "status:\n%s", st.String())
}

func TestStatus_ShowsTrackedFileEvenWhenIgnored(t *testing.T) {
	_ = gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "foo", "old\n")
	gitfixture.CommitFile(t, cx, ".gitignore", "foo\n")
	gitfixture.WriteRepoFile(t, cx, "foo", "new\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.Equal(t, git.Modified, requireStatus(t, st, "foo").Worktree)
	assert.Equal(t, git.Unmodified, requireStatus(t, st, "foo").Staging)
}

func TestStatus_LocalCoreExcludesFileOverridesGlobal(t *testing.T) {
	home := gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	globalIgnore := home("global-ignore")
	gitfixture.WriteFile(t, globalIgnore, "globalonly/\n")
	gitfixture.WriteFile(t, home(".gitconfig"),
		"[core]\n\texcludesFile = "+globalIgnore+"\n")

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")

	localIgnore := home("local-ignore")
	gitfixture.WriteFile(t, localIgnore, "localonly/\n")

	dotGit, ok := cx.DotGitPath()
	require.True(t, ok)
	gitfixture.WriteFile(t, filepath.Join(dotGit, "config"),
		"[core]\n\texcludesFile = "+localIgnore+"\n")

	gitfixture.WriteRepoFile(t, cx, "globalonly/file", "visible\n")
	gitfixture.WriteRepoFile(t, cx, "localonly/file", "ignored\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.Equal(t, git.Untracked, requireStatus(t, st, "globalonly/file").Worktree)
	assert.NotContains(t, st, "localonly/file")
}

func TestStatus_HonorsCoreIgnoreCase(t *testing.T) {
	_ = gitfixture.NewHomeFixture(t)
	gitfixture.IsolateGitConfig(t)

	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "tracked", "tracked\n")
	gitfixture.CommitFile(t, cx, ".gitignore", "BUILD/\n")

	dotGit, ok := cx.DotGitPath()
	require.True(t, ok)
	gitfixture.WriteFile(t, filepath.Join(dotGit, "config"),
		"[core]\n\tignoreCase = true\n")

	gitfixture.WriteRepoFile(t, cx, "build/file", "ignored\n")

	st, err := gitrepo.BuildWorktreeStatus(cx)
	require.NoError(t, err)
	assert.NotContains(t, st, "build/file")
	assert.True(t, st.IsClean(), "status:\n%s", st.String())
}
