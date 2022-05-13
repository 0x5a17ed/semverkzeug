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

package gitversions

import (
	"errors"
	"time"

	"github.com/go-git/go-git/v5"
)

var (
	ErrNotDirty = errors.New("repository is clean")
)

func LastModificationTime(wt *git.Worktree, st git.Status) (*time.Time, error) {
	var latestChange time.Time

	for fp, fs := range st {
		if fs.Worktree == git.Unmodified && fs.Staging == git.Unmodified {
			continue
		}

		if fi, err := wt.Filesystem.Lstat(fp); err == nil {
			if mtime := fi.ModTime(); mtime.After(latestChange) {
				latestChange = mtime.UTC()
			}
		} else {
			return nil, err
		}
	}

	if latestChange.IsZero() {
		return nil, ErrNotDirty
	}
	return &latestChange, nil
}
