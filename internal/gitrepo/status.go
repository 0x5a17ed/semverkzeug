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

package gitrepo

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

// BuildWorktreeStatus computes the worktree status of the current repository
// context by comparing the working tree state with the index. It returns a
// [git.Status] or an error.
func BuildWorktreeStatus(cx *Context) (git.Status, error) {
	wt, err := cx.LoadWorktree()
	if err != nil {
		return nil, fmt.Errorf("load worktree: %w", err)
	}

	st, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("build status: %w", err)
	}

	return st, err
}
