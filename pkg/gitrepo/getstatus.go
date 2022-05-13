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

package gitrepo

import (
	"github.com/go-git/go-git/v5"
)

// GetStatus returns the working tree with its status with external
// excludes patterns loaded from the `exlcudesfile` setting found in
// the global git configuration file.
func GetStatus(r *git.Repository) (wt *git.Worktree, st git.Status, err error) {
	if wt, err = r.Worktree(); err != nil {
		return
	}

	ignorePatterns, err := loadGlobalExcludePatterns()
	if err != nil {
		return
	}
	wt.Excludes = append(wt.Excludes, ignorePatterns...)

	st, err = wt.Status()
	return
}
