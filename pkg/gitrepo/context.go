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
	"net/url"
	"sync"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Context struct {
	repo *git.Repository

	wt struct {
		once  sync.Once
		value *git.Worktree
		err   error
	}
}

func (cx *Context) loadWorktreeOnce() (*git.Worktree, error) {
	wt, err := cx.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("get worktree: %w", err)
	}

	rootFS := osfs.New("/")

	systemPatterns, err := gitignore.LoadSystemPatterns(rootFS)
	if err != nil {
		return nil, fmt.Errorf("load system gitignore patterns: %w", err)
	}
	wt.Excludes = append(wt.Excludes, systemPatterns...)

	globalPatterns, err := gitignore.LoadGlobalPatterns(rootFS)
	if err != nil {
		return nil, fmt.Errorf("load global gitignore patterns: %w", err)
	}
	wt.Excludes = append(wt.Excludes, globalPatterns...)

	return wt, nil
}

func (cx *Context) String() string {
	var repoString string
	if cx.repo == nil {
		repoString = "<nil>"
	} else if st, ok := cx.repo.Storer.(*filesystem.Storage); ok {
		repoString = fmt.Sprintf("<filesystem:path=%s>", new(url.URL{Path: st.Filesystem().Root()}).String())
	} else if _, ok := cx.repo.Storer.(*memory.Storage); ok {
		repoString = "<memory>"
	} else {
		repoString = "<unknown>"
	}

	return fmt.Sprintf("gitrepo.Context{repo=%s}", repoString)
}

// Repository returns the underlying git repository.
func (cx *Context) Repository() *git.Repository {
	return cx.repo
}

// LoadWorktree returns a worktree for the repository.
func (cx *Context) LoadWorktree() (*git.Worktree, error) {
	cx.wt.once.Do(func() {
		cx.wt.value, cx.wt.err = cx.loadWorktreeOnce()
	})

	return cx.wt.value, cx.wt.err
}

// LoadWorktreeFilesystem returns the filesystem for the repository.
func (cx *Context) LoadWorktreeFilesystem() (billy.Filesystem, error) {
	wt, err := cx.LoadWorktree()
	if err != nil {
		return nil, fmt.Errorf("load worktree: %w", err)
	}
	return wt.Filesystem, nil
}

// NewContextFromRepo creates a new Context instance from the provided git repository.
func NewContextFromRepo(r *git.Repository) (*Context, error) {
	cx := &Context{
		repo: r,
	}

	return cx, nil
}

// NewContextFromPath creates a new Context from a git repository at the given
// path on the filesystem.
func NewContextFromPath(p string) (*Context, error) {
	repo, err := git.PlainOpenWithOptions(p, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open repository %#q: %w", p, err)
	}

	return NewContextFromRepo(repo)
}
