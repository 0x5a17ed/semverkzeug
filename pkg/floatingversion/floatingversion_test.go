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

package floatingversion_test

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/floatingversion"
	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/internal/gitfixture"
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
			cx := tt.args.repo(t)

			head, err := cx.Repository().Head()
			if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
				require.NoError(t, err)
			}

			gotVs, err := floatingversion.Describe(cx, head, gitrepo.RootScope())
			require.NoError(t, err)

			assert.Regexp(t, tt.args.wantVer, gotVs.String())
		})
	}
}
