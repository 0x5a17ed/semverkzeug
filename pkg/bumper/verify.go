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

package bumper

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

// VerifyRepo validates the repository state by checking the reference and
// ensuring a clean working tree status.
func VerifyRepo(cx *gitrepo.Context, ref *plumbing.Reference) error {
	if ref == nil {
		return ErrRepositoryIsEmpty
	}

	st, err := gitrepo.BuildWorktreeStatus(cx)
	if err != nil {
		return fmt.Errorf("read worktree status: %w", err)
	}

	if !st.IsClean() {
		return ErrRepositoryIsDirty
	}

	return nil
}
