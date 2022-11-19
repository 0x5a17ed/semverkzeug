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
	"io/fs"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
)

var (
	ErrNotDirty = errors.New("repository is clean")
)

func findModTime(vfs billy.Filesystem, fp string) (time.Time, error) {
	for {
		fi, err := vfs.Lstat(fp)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && fp != "." {
				// File was probably deleted. Try parent directory.
				fp = filepath.Dir(fp)
				continue
			}
			return time.Time{}, err
		}
		return fi.ModTime().UTC(), nil
	}
}

func LastModificationTime(wt *git.Worktree, st git.Status) (*time.Time, error) {
	var latestChange time.Time

	for fp, fst := range st {
		if fst.Worktree == git.Unmodified && fst.Staging == git.Unmodified {
			continue
		}

		switch mtime, err := findModTime(wt.Filesystem, fp); {
		case err != nil:
			return nil, err
		case mtime.After(latestChange):
			latestChange = mtime
		}
	}

	if latestChange.IsZero() {
		return nil, ErrNotDirty
	}
	return &latestChange, nil
}
