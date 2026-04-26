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
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// GetStatus returns the working tree with its status with external
// exclude patterns loaded from the system and global git
// configuration files, mimicking native git behavior.
func GetStatus(r *git.Repository) (wt *git.Worktree, st git.Status, err error) {
	if wt, err = r.Worktree(); err != nil {
		return
	}

	rootFS := osfs.New("/")

	systemPatterns, err := gitignore.LoadSystemPatterns(rootFS)
	if err != nil {
		return
	}
	wt.Excludes = append(wt.Excludes, systemPatterns...)

	globalPatterns, err := gitignore.LoadGlobalPatterns(rootFS)
	if err != nil {
		return
	}
	wt.Excludes = append(wt.Excludes, globalPatterns...)

	st, err = wt.Status()
	return
}
