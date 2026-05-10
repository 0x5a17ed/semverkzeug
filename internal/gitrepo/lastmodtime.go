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
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"path/filepath"
	"time"

	"github.com/0x5a17ed/xit"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
)

var (
	ErrWorktreeClean = errors.New("repository worktree is clean")
)

type findMTimeResult struct {
	mtime time.Time
	next  string
	found bool
}

func findMTimeStep(vfs billy.Filesystem, fp string) (findMTimeResult, error) {
	fi, err := vfs.Lstat(fp)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// File was deleted. Try the parent directory.
		parent := filepath.Dir(fp)

		// Make sure we haven't reached the root yet.
		if parent == fp {
			return findMTimeResult{}, fmt.Errorf("stat %q: %w", fp, err)
		}

		// Force a directory read to bust the NFS attribute cache.
		// Reading entries often triggers fresher metadata than Lstat alone.
		_, _ = vfs.ReadDir(parent)

		// Move up to the parent and try again.
		return findMTimeResult{
			next: parent,
		}, nil

	case err != nil:
		return findMTimeResult{}, fmt.Errorf("stat %q: %w", fp, err)
	}

	return findMTimeResult{
		mtime: fi.ModTime().UTC(),
		found: true,
	}, nil
}

// findMTimePath returns the modification time associated with fp in the working tree.
//
// For existing paths, it returns the path's own mtime.
//
// For paths that have been deleted, it walks up to the nearest existing
// ancestor and returns that ancestor's mtime. This lets callers still derive
// a meaningful modification time for deleted entries, since removing a file
// updates its parent directory rather than leaving metadata on the removed
// path itself.
//
// A directory read may be performed before restatting an ancestor to reduce
// stale metadata from filesystem attribute caches.
//
// The returned time is normalized to UTC.
func findMTimePath(vfs billy.Filesystem, fp string) (time.Time, error) {
	for {
		result, err := findMTimeStep(vfs, fp)
		if err != nil {
			return time.Time{}, err
		}

		if result.found {
			return result.mtime, nil
		}

		fp = result.next
	}
}

// findIndexMTime returns the modification time of the repository's
// index file when the repo is backed by filesystem storage.  Returns
// a nil time without error for in-memory storage or when the index
// has not yet been created.
func findIndexMTime(cx *Context) (*time.Time, error) {
	fsys := cx.DotGitFilesystem()
	if fsys == nil {
		return nil, nil
	}

	fi, err := fsys.Stat("index")
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("stat index: %w", err)
	}

	return new(fi.ModTime().UTC()), nil
}

// DirtyEntry describes a single entry in the worktree status that is not
// fully unmodified, together with the effective mtime resolved for it.
type DirtyEntry struct {
	path     string
	worktree git.StatusCode
	staging  git.StatusCode
	mtime    time.Time
}

func (e *DirtyEntry) Path() string                   { return e.path }
func (e *DirtyEntry) WorktreeStatus() git.StatusCode { return e.worktree }
func (e *DirtyEntry) StagingStatus() git.StatusCode  { return e.staging }
func (e *DirtyEntry) ModTime() time.Time             { return e.mtime }

// IterDirtyEntries walks the worktree status and returns the dirty entries
// with their effective mtimes (using the deleted-ancestor walk for paths
// that no longer exist), along with the index file's mtime if available.
func IterDirtyEntries(cx *Context) (iter.Seq[DirtyEntry], func() error) {
	return xit.Perform(func(yield func(DirtyEntry) bool) error {
		wtFsys, err := cx.LoadWorktreeFilesystem()
		if err != nil {
			return fmt.Errorf("load worktree: %w", err)
		}

		st, err := BuildWorktreeStatus(cx)
		if err != nil {
			return fmt.Errorf("build worktree status: %w", err)
		}

		for fp, fst := range st {
			if fst.Worktree == git.Unmodified && fst.Staging == git.Unmodified {
				continue
			}

			mtime, err := findMTimePath(wtFsys, fp)
			if err != nil {
				return fmt.Errorf("find mtime for %q: %w", fp, err)
			}

			if !yield(DirtyEntry{
				path:     fp,
				worktree: fst.Worktree,
				staging:  fst.Staging,
				mtime:    mtime,
			}) {
				return nil
			}
		}
		return nil
	})
}
