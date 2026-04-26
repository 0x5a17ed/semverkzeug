/*
 * Copyright(C) 2022 individual contributors
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

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/testhelper"
)

func mustScope(t *testing.T, s string) gitrepo.Scope {
	t.Helper()
	sc, err := gitrepo.ParseScope(s)
	require.NoError(t, err)
	return sc
}

func taggedFixture(t *testing.T) (*testhelper.Repo, *plumbing.Reference) {
	scope := testhelper.RepoOneCommitClean(t)

	head, err := scope.Repo.Head()
	require.NoError(t, err)

	_, err = scope.Repo.CreateTag("v1.0.0", head.Hash(), nil)
	require.NoError(t, err)

	testhelper.CommitFile(t, scope, "baa", "baz")

	_, err = scope.Repo.CreateTag("mod/v1.0.0", head.Hash(), nil)
	require.NoError(t, err)

	return scope, head
}

func TestLatestScopedTagSelection(t *testing.T) {
	repo, head := taggedFixture(t)

	t.Run("root-scope", func(t *testing.T) {
		vs, err := gitrepo.FindLatestVersion(repo.Repo, head, mustScope(t,"."))
		require.NoError(t, err)

		require.Len(t, vs.Guide.Tags, 1)
		assert.Equal(t, "v1.0.0", vs.Guide.Tags[0].TagName)
		assert.Equal(t, "v", vs.Spec.Prefix)
		assert.Equal(t, "1.0.0", vs.Spec.Version.String())
	})

	t.Run("module-scope", func(t *testing.T) {
		vs, err := gitrepo.FindLatestVersion(repo.Repo, head, mustScope(t,"mod"))
		require.NoError(t, err)

		require.Len(t, vs.Guide.Tags, 1)
		assert.Equal(t, "mod/v1.0.0", vs.Guide.Tags[0].TagName)
		assert.Equal(t, "mod", vs.Spec.Scope.String())
		assert.Equal(t, "v", vs.Spec.Prefix)
		assert.Equal(t, "1.0.0", vs.Spec.Version.String())
	})
}
