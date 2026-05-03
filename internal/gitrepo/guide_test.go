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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

func mustScope(t *testing.T, s string) gitrepo.Scope {
	t.Helper()

	sc, err := gitrepo.ParseScope(s)
	require.NoError(t, err)

	return sc
}

func TestBuildGuide_ScopedTagSelection(t *testing.T) {
	cx := gitfixture.RepoWithScopedTags(t)

	head := gitfixture.Head(t, cx)

	t.Run("root-scope", func(t *testing.T) {
		guide, err := gitrepo.BuildGuide(cx, head, gitrepo.RootScope())
		require.NoError(t, err)

		vt := guide.HighestVersion()
		require.NotNil(t, vt)
		assert.Equal(t, "v1.0.0", vt.TagName)
		assert.Equal(t, "v", vt.VersionSpec.Prefix)
		assert.Equal(t, "1.0.0", vt.VersionSpec.Version.String())
	})

	t.Run("module-scope", func(t *testing.T) {
		guide, err := gitrepo.BuildGuide(cx, head, mustScope(t, "mod"))
		require.NoError(t, err)

		vt := guide.HighestVersion()
		require.NotNil(t, vt)
		assert.Equal(t, "mod/v2.0.0", vt.TagName)
		assert.Equal(t, "mod", vt.VersionSpec.Scope.String())
		assert.Equal(t, "v", vt.VersionSpec.Prefix)
		assert.Equal(t, "2.0.0", vt.VersionSpec.Version.String())
	})
}

// TestBuildGuide_SideBranchTag covers the release-branch
// model where the version tag lives on a branch cut from main.
//
//	A--B--C (main, HEAD)
//	    \
//	     T [v0.5.0] (release/0.5)
//
// merge-base(C, T) == B; main is one commit ahead of where v0.5
// branched.
func TestBuildGuide_SideBranchTag(t *testing.T) {
	repo := gitfixture.RepoWithOneCommitNoTagsClean(t) // commit A on main
	bCommit := gitfixture.CommitFile(t, repo, "b.txt", "b")

	gitfixture.Checkout(t, repo, "release/0.5", true)
	tCommit := gitfixture.CommitFile(t, repo, "release.txt", "notes")
	_, err := repo.Repository().CreateTag("v0.5.0", tCommit.Hash, nil)
	require.NoError(t, err)

	gitfixture.Checkout(t, repo, "main", false)
	c := gitfixture.CommitFile(t, repo, "c.txt", "c")

	head, err := repo.Repository().Head()
	require.NoError(t, err)
	require.Equal(t, c.Hash, head.Hash())

	guide, err := gitrepo.BuildGuide(repo, head, gitrepo.RootScope())
	require.NoError(t, err)

	vt := guide.HighestVersion()
	require.NotNil(t, vt)
	assert.Equal(t, "v0.5.0", vt.TagName)
	assert.Equal(t, "0.5.0", vt.VersionSpec.Version.String())
	require.NotNil(t, guide.MergeBase)
	assert.Equal(t, bCommit.Hash, guide.MergeBase.Hash)
	assert.Equal(t, 1, guide.Depth)
}

// TestBuildGuide_MultipleReleaseBranches mirrors the
// brainstorm scenario: two release branches diverge from different
// points on main.
//
//	A--B--C--D (main, HEAD)
//	    \  \
//	     \  t05 [v0.5.0]
//	      t04 [v0.4.0]
//
// v0.5.0 wins by semver; merge-base(D, v0.5.0) == C.
func TestBuildGuide_MultipleReleaseBranches(t *testing.T) {
	repo := gitfixture.RepoWithOneCommitNoTagsClean(t) // A
	bCommit := gitfixture.CommitFile(t, repo, "b.txt", "b")
	cCommit := gitfixture.CommitFile(t, repo, "c.txt", "c")

	// release/0.4 branches from B.
	require.NoError(t, gitfixture.Worktree(t, repo).Checkout(&git.CheckoutOptions{
		Hash:   bCommit.Hash,
		Branch: plumbing.NewBranchReferenceName("release/0.4"),
		Create: true,
	}))
	t04 := gitfixture.CommitFile(t, repo, "rel04.txt", "0.4 notes")
	_, err := repo.Repository().CreateTag("v0.4.0", t04.Hash, nil)
	require.NoError(t, err)

	// release/0.5 branches from C.
	gitfixture.Checkout(t, repo, "main", false)
	gitfixture.Checkout(t, repo, "release/0.5", true)
	t05 := gitfixture.CommitFile(t, repo, "rel05.txt", "0.5 notes")
	_, err = repo.Repository().CreateTag("v0.5.0", t05.Hash, nil)
	require.NoError(t, err)

	// One more commit on main.
	gitfixture.Checkout(t, repo, "main", false)
	dCommit := gitfixture.CommitFile(t, repo, "d.txt", "d")

	head, err := repo.Repository().Head()
	require.NoError(t, err)
	require.Equal(t, dCommit.Hash, head.Hash())

	guide, err := gitrepo.BuildGuide(repo, head, gitrepo.RootScope())
	require.NoError(t, err)

	require.Len(t, guide.Tags, 1)
	assert.Equal(t, "v0.5.0", guide.Tags[0].TagName)
	require.NotNil(t, guide.MergeBase)
	assert.Equal(t, cCommit.Hash, guide.MergeBase.Hash)
	assert.Equal(t, 1, guide.Depth)
}

// TestBuildGuide_AncestorOfTag verifies that a tag living in
// HEAD's "future" (HEAD is a strict ancestor of the tag) is not
// selected.  Otherwise, we'd report a version that didn't exist at
// HEAD's point in history.
func TestBuildGuide_AncestorOfTag(t *testing.T) {
	repo := gitfixture.RepoWithOneCommitNoTagsClean(t)

	headRef, err := repo.Repository().Head()
	require.NoError(t, err)
	earlyHash := headRef.Hash()

	// Add later commits and tag the latest one.
	gitfixture.CommitFile(t, repo, "b.txt", "b")
	tip := gitfixture.CommitFile(t, repo, "c.txt", "c")
	_, err = repo.Repository().CreateTag("v0.5.0", tip.Hash, nil)
	require.NoError(t, err)

	// Look up version from the early commit's perspective — v0.5.0
	// hasn't been "released" yet from there.
	earlyRef := plumbing.NewHashReference(plumbing.HEAD, earlyHash)

	guide, err := gitrepo.BuildGuide(repo, earlyRef, gitrepo.RootScope())
	require.NoError(t, err)

	assert.Empty(t, guide.Tags, "tag in HEAD's future must not be picked")
	assert.Nil(t, guide.MergeBase)
}

// TestBuildGuide_StrandedTagFiltered verifies that a tag on a
// commit not reachable from any branch is ignored — the merge-base
// strategy would otherwise match it via the repo root.
func TestBuildGuide_StrandedTagFiltered(t *testing.T) {
	repo := gitfixture.RepoWithOneCommitNoTagsClean(t) // A on main

	// Build a side branch with a tag, then delete the branch so the
	// tag itself only keeps the tagged commit alive.
	gitfixture.Checkout(t, repo, "experimental", true)
	expCommit := gitfixture.CommitFile(t, repo, "exp.txt", "experiment")
	_, err := repo.Repository().CreateTag("v9.0.0", expCommit.Hash, nil)
	require.NoError(t, err)

	gitfixture.Checkout(t, repo, "main", false)
	require.NoError(t, repo.Repository().Storer.RemoveReference(
		plumbing.NewBranchReferenceName("experimental"),
	))

	// Add a real release tag on main.
	mainHead, err := repo.Repository().Head()
	require.NoError(t, err)
	_, err = repo.Repository().CreateTag("v0.1.0", mainHead.Hash(), nil)
	require.NoError(t, err)

	guide, err := gitrepo.BuildGuide(repo, mainHead, gitrepo.RootScope())
	require.NoError(t, err)

	require.Len(t, guide.Tags, 1)
	assert.Equal(t, "v0.1.0", guide.Tags[0].TagName,
		"stranded v9.0.0 must not outrank reachable v0.1.0")
}

// TestFindLatestVersionTagOnRemoteTrackingBranch covers the typical
// CI / fresh-clone setup: only the checked-out branch exists locally;
// every other branch is present as `refs/remotes/origin/*`.  A
// release-branch tag whose commit is reachable solely via a
// remote-tracking ref is *not* stranded and must still be selected.
//
//	A--B--C (main, HEAD)
//	    \
//	     T [v0.5.0] (origin/release/0.5; no local release branch)
func TestBuildGuide_RemoteTrackingBranch(t *testing.T) {
	// Arrange: build the side branch locally, tag it, then convert it
	// into a remote-tracking-only ref so no `refs/heads/*` reaches T.
	repo := gitfixture.RepoWithOneCommitNoTagsClean(t) // A on main
	bCommit := gitfixture.CommitFile(t, repo, "b.txt", "b")

	gitfixture.Checkout(t, repo, "release/0.5", true)
	tCommit := gitfixture.CommitFile(t, repo, "release.txt", "notes")
	_, err := repo.Repository().CreateTag("v0.5.0", tCommit.Hash, nil)
	require.NoError(t, err)

	gitfixture.Checkout(t, repo, "main", false)
	require.NoError(t, repo.Repository().Storer.RemoveReference(
		plumbing.NewBranchReferenceName("release/0.5"),
	))
	require.NoError(t, repo.Repository().Storer.SetReference(
		plumbing.NewHashReference(
			plumbing.NewRemoteReferenceName("origin", "release/0.5"),
			tCommit.Hash,
		),
	))

	c := gitfixture.CommitFile(t, repo, "c.txt", "c")

	head, err := repo.Repository().Head()
	require.NoError(t, err)
	require.Equal(t, c.Hash, head.Hash())

	// Act.
	guide, err := gitrepo.BuildGuide(repo, head, gitrepo.RootScope())
	require.NoError(t, err)

	// Assert: v0.5.0 is reachable via origin/release/0.5 and must not
	// be filtered out as stranded.
	require.Len(t, guide.Tags, 1,
		"v0.5.0 reachable via origin/release/0.5 must not be filtered as stranded")
	assert.Equal(t, "v0.5.0", guide.Tags[0].TagName)
	require.NotNil(t, guide.MergeBase)
	assert.Equal(t, bCommit.Hash, guide.MergeBase.Hash)
	assert.Equal(t, 1, guide.Depth)
}
