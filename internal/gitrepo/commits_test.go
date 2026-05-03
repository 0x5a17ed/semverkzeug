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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/internal/gitfixture"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

func TestAbbreviatedCommitHash(t *testing.T) {
	repo := gitfixture.RepoEmpty(t)

	c := gitfixture.CommitFile(t, repo, "test.txt", "test")

	got, err := gitrepo.FindUniqueCommitHashAbbreviation(repo, c)
	require.NoError(t, err)

	assert.Equal(t, c.Hash.String()[:7], got)
}
