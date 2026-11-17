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

package bumper_test

import (
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/bumper"
	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

func gitEnvFixture(t *testing.T) {
	t.Helper()

	t.Setenv("GIT_AUTHOR_NAME", gitfixture.TestSig.Name)
	t.Setenv("GIT_AUTHOR_EMAIL", gitfixture.TestSig.Email)
	t.Setenv("GIT_COMMITTER_NAME", gitfixture.TestSig.Name)
	t.Setenv("GIT_COMMITTER_EMAIL", gitfixture.TestSig.Email)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), ".gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "tag.gpgSign")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")
}

func TestCreateTag(t *testing.T) {
	t.Run("filesystem-empty", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoEmpty(t)

		gitEnvFixture(t)

		// Act
		_, err := bumper.CreateTag(cx, nil, bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{})

		// Assert
		assert.ErrorIs(t, err, bumper.ErrRepositoryIsEmpty)
	})

	t.Run("filesystem-first-release", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitNoTagsClean(t)

		gitEnvFixture(t)

		// Act
		tagRef, err := bumper.CreateTag(cx, gitfixture.Head(t, cx), bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{})

		// Assert
		require.NoError(t, err)

		assert.Equal(t, plumbing.NewTagReferenceName("v0.0.1"), tagRef.Name())

		resolvedRef, err := cx.Repository().Tag("v0.0.1")
		require.NoError(t, err)
		assert.Equal(t, tagRef.Hash(), resolvedRef.Hash())

		tagObject, err := cx.Repository().TagObject(tagRef.Hash())
		require.NoError(t, err)
		assert.Equal(t, "v0.0.1", tagObject.Name)
		assert.Equal(t, "first version v0.0.1\n", tagObject.Message)
	})

	t.Run("filesystem-populated", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitOneTagClean(t)

		gitEnvFixture(t)

		// Act
		tagRef, err := bumper.CreateTag(cx, gitfixture.Head(t, cx), bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{})

		// Assert
		require.NoError(t, err)

		assert.Equal(t, plumbing.NewTagReferenceName("v0.1.1"), tagRef.Name())

		resolvedRef, err := cx.Repository().Tag("v0.1.1")
		require.NoError(t, err)
		assert.Equal(t, tagRef.Hash(), resolvedRef.Hash())

		tagObject, err := cx.Repository().TagObject(tagRef.Hash())
		require.NoError(t, err)
		assert.Equal(t, "v0.1.1", tagObject.Name)
		assert.Equal(t, "bump version v0.1.0 -> v0.1.1\n", tagObject.Message)
	})
}

func TestCreateTagOptions(t *testing.T) {
	t.Run("expect-matching-version", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitOneTagClean(t)
		gitEnvFixture(t)
		expect := semver.MustParse("0.1.1")

		// Act
		tagRef, err := bumper.CreateTag(cx, gitfixture.Head(t, cx), bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{Expect: expect})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, plumbing.NewTagReferenceName("v0.1.1"), tagRef.Name())
	})

	t.Run("expect-other-version", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitOneTagClean(t)
		gitEnvFixture(t)
		expect := semver.MustParse("0.2.0")

		// Act
		_, err := bumper.CreateTag(cx, gitfixture.Head(t, cx), bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{Expect: expect})

		// Assert
		assert.ErrorIs(t, err, bumper.ErrUnexpectedVersion)
		assert.ErrorContains(t, err, "bumping v0.1.0 yields v0.1.1, not 0.2.0")
		_, err = cx.Repository().Tag("v0.1.1")
		assert.ErrorIs(t, err, git.ErrTagNotFound)
	})

	t.Run("message-keeps-markdown-headings", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitOneTagClean(t)
		gitEnvFixture(t)
		message := "## [`0.1.1`] (2026-11-17)\n\n### Fixed\n\n* Crash on start\n"

		// Act
		tagRef, err := bumper.CreateTag(cx, gitfixture.Head(t, cx), bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{Message: message})

		// Assert
		require.NoError(t, err)
		tagObject, err := cx.Repository().TagObject(tagRef.Hash())
		require.NoError(t, err)
		assert.Equal(t, message, tagObject.Message)
	})
}

func TestPlanTag(t *testing.T) {
	t.Run("creates-nothing", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitOneTagClean(t)
		gitEnvFixture(t)

		// Act
		plan, err := bumper.PlanTag(cx, gitfixture.Head(t, cx), bumper.Minor, gitrepo.RootScope(), bumper.TagOptions{})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "v0.2.0", plan.Label)
		assert.Equal(t, "bump version v0.1.0 -> v0.2.0", plan.Message)
		_, err = cx.Repository().Tag("v0.2.0")
		assert.ErrorIs(t, err, git.ErrTagNotFound)
	})

	t.Run("dirty-repository", func(t *testing.T) {
		// Arrange
		cx := gitfixture.RepoWithOneCommitOneTagDirty(t)
		gitEnvFixture(t)

		// Act
		_, err := bumper.PlanTag(cx, gitfixture.Head(t, cx), bumper.Patch, gitrepo.RootScope(), bumper.TagOptions{})

		// Assert
		assert.ErrorIs(t, err, bumper.ErrRepositoryIsDirty)
	})
}
