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

package floatingversion_test

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/floatingversion"
	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

type repositoryProvider func(t *testing.T) *gitrepo.Context

func TestDescribe(t *testing.T) {
	type args struct {
		repo    repositoryProvider
		wantVer string
	}

	tests := []struct {
		name string
		args args
	}{
		{"empty", args{repo: gitfixture.RepoEmpty, wantVer: `v0\.0\.1-dev\.0`}},
		{"empty-dirty", args{repo: gitfixture.RepoWithNoCommitsNoTagsDirty, wantVer: `v0\.0\.1-dev\.\d{6}T\d{8}Z`}},
		{"one-commit-no-tag-clean", args{repo: gitfixture.RepoWithOneCommitNoTagsClean, wantVer: `v0\.0\.1-dev\.\d{6}T\d{8}Z`}},
		{"one-commit-no-tag-dirty", args{repo: gitfixture.RepoWithOneCommitNoTagsDirty, wantVer: `v0\.0\.1-dev\.\d{6}T\d{8}Z`}},
		{"one-commit-no-tag-file-deleted", args{repo: gitfixture.RepoWithOneCommitNoTagsFileDeleted, wantVer: `v0\.0\.1-dev\.\d{6}T\d{8}Z`}},
		{"one-tag-clean", args{repo: gitfixture.RepoWithOneCommitOneTagClean, wantVer: `v0\.1\.0`}},
		{"one-tag-dirty", args{repo: gitfixture.RepoWithOneCommitOneTagDirty, wantVer: `v0\.1\.1-dev\.\d{6}T\d{8}Z`}},
		{"one-one-commit-past-one-tag-clean", args{repo: gitfixture.RepoWithTwoCommitsOneTagClean, wantVer: `v0\.1\.1-dev\.\d{6}T\d{8}Z`}},
		{"one-one-commit-past-one-tag-dirty", args{repo: gitfixture.RepoWithTwoCommitsOneTagDirty, wantVer: `v0\.1\.1-dev\.\d{6}T\d{8}Z`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Set up the test environment and fixtures.
			cx := tt.args.repo(t)

			head, err := cx.Repository().Head()
			if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
				require.NoError(t, err)
			}

			guide, err := gitrepo.BuildGuide(cx, head, gitrepo.RootScope())
			require.NoError(t, err)

			// Act: Describe the floating version based on the guide.
			gotVs, err := floatingversion.Describe(cx, guide)
			require.NoError(t, err)

			// Assert: The floating version matches the expected pattern.
			assert.Regexp(t, tt.args.wantVer, gotVs.String())
		})
	}
}

// TestDescribe_DirtyPrereleaseTagMustSortAbove ensures a dev snapshot
// taken on top of a pre-release tag (e.g. v1.0.0-rc.1) sorts strictly
// above the tag it derives from.  Replacing the rc.1 prerelease with
// "dev.<mtime>" yields v1.0.0-dev.<mtime>, which sorts below
// v1.0.0-rc.1 because semver compares prerelease identifiers
// lexically and "dev" < "rc".  The dev build would then masquerade
// as older than the release candidate.
func TestDescribe_DirtyPrereleaseTagMustSortAbove(t *testing.T) {
	// Arrange: Build a repo whose HEAD is exactly v1.0.0-rc.1 and
	// dirty the worktree so Describe takes the dev-snapshot path.
	cx := gitfixture.RepoEmpty(t)
	gitfixture.CommitFile(t, cx, "foo", "baa")
	gitfixture.CreateTag(t, cx, "v1.0.0-rc.1")
	gitfixture.WriteRepoFile(t, cx, "foo", "baz")

	head, err := cx.Repository().Head()
	require.NoError(t, err)

	guide, err := gitrepo.BuildGuide(cx, head, gitrepo.RootScope())
	require.NoError(t, err)

	srcTag := semver.MustParse("1.0.0-rc.1")

	// Act: Describe the floating version on top of the dirty rc.
	gotVs, err := floatingversion.Describe(cx, guide)
	require.NoError(t, err)

	// Assert: The dev snapshot must sort strictly above the source
	// tag; otherwise the snapshot appears older than the rc itself.
	assert.Truef(t,
		gotVs.Version.GreaterThan(srcTag),
		"dev snapshot %q must sort above source tag %q", gotVs.String(), srcTag,
	)
}

// TestDescribe_DirtyDevPrereleaseReplacesCounter pins the
// replace-don't-accumulate behaviour for the dev.<n> case: when the
// source tag's prerelease already carries a dev counter, Describe
// must swap it for a fresh dev.<mtime> rather than appending — which
// would yield e.g. v1.0.0-dev.5.dev.<mtime> and grow on every dirty
// build.
func TestDescribe_DirtyDevPrereleaseReplacesCounter(t *testing.T) {
	type args struct {
		tag       string
		wantRegex string
	}

	tests := []struct {
		name string
		args args
	}{
		{"pure-dev-counter", args{tag: "v1.0.0-dev.5", wantRegex: `^v1\.0\.0-dev\.\d{6}T\d{8}Z$`}},
		{"embedded-dev-counter", args{tag: "v1.0.0-alpha.dev.3", wantRegex: `^v1\.0\.0-alpha\.dev\.\d{6}T\d{8}Z$`}},

		{"identifier-boundary-1", args{tag: "v1.0.0-predev.3", wantRegex: `^v1\.0\.0-predev\.3\.dev\.\d{6}T\d{8}Z$`}},
		{"identifier-boundary-2", args{tag: "v1.0.0-developer.1", wantRegex: `^v1\.0\.0-developer\.1\.dev\.\d{6}T\d{8}Z$`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Tag HEAD with the source prerelease and dirty
			// the worktree so Describe takes the dev-snapshot path.
			cx := gitfixture.RepoEmpty(t)
			gitfixture.CommitFile(t, cx, "foo", "baa")
			gitfixture.CreateTag(t, cx, tt.args.tag)
			gitfixture.WriteRepoFile(t, cx, "foo", "baz")

			head, err := cx.Repository().Head()
			require.NoError(t, err)

			guide, err := gitrepo.BuildGuide(cx, head, gitrepo.RootScope())
			require.NoError(t, err)

			// Act: Describe the floating version on top of the dirty tag.
			gotVs, err := floatingversion.Describe(cx, guide)
			require.NoError(t, err)

			// Assert: The existing dev.<n> counter is replaced (not
			// accumulated) by a fresh dev.<mtime>, with the
			// preceding prerelease identifiers preserved verbatim.
			assert.Regexp(t, tt.args.wantRegex, gotVs.String())
		})
	}
}
