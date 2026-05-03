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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
)

// scopeForRepoPath resolves p into a tag scope relative to the
// repository worktree root.
//
// It returns the root scope for root-scoped operation, or when no
// worktree is available.
func scopeForRepoPath(repo *gitrepo.Context, p string) (gitrepo.Scope, error) {
	if p == "" {
		return gitrepo.RootScope(), nil
	}

	// Resolve the user input to an absolute path so that the following
	// path math is stable regardless of the current working directory.
	absPath, err := filepath.Abs(p)
	if err != nil {
		return gitrepo.Scope{}, fmt.Errorf("resolve absolute path: %w", err)
	}

	// Discover the repository root from the checked-out worktree.
	// If no worktree is available, fall back to the root scope.
	wt, err := repo.LoadWorktree()
	if err != nil {
		return gitrepo.RootScope(), nil
	}

	// Normalize the worktree root to an absolute path to keep comparison
	// logic consistent with absPath above.
	rootPath, err := filepath.Abs(wt.Filesystem.Root())
	if err != nil {
		return gitrepo.Scope{}, err
	}

	// Convert the target path into a path relative to the repository root.
	// This relative segment becomes the tag scope (for example, a submodule).
	relPath, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		return gitrepo.Scope{}, err
	}

	// Convert separators to "/" so scope values are platform-independent.
	relPath = filepath.ToSlash(relPath)

	return gitrepo.ParseScope(relPath)
}

// provideRepo opens the git repository pointed at by --repo (or $PWD
// when --repo is unset).  It is registered with kong as a singleton
// provider so that multiple Run() arguments resolve to the same
// gitrepo.Context per invocation.
func provideRepo(root *cli) (*gitrepo.Context, error) {
	repoPath := root.Repo
	if repoPath == "" {
		var err error
		if repoPath, err = os.Getwd(); err != nil {
			return nil, err
		}
	}

	cx, err := gitrepo.NewContextFromPath(repoPath)
	if err != nil {
		return nil, fmt.Errorf("create git context: %w", err)
	}
	return cx, nil
}

// provideHead resolves the repository's HEAD reference.  An empty
// repository (HEAD missing) is reported as a nil reference rather
// than an error so commands can decide how to react.
func provideHead(repo *gitrepo.Context) (*plumbing.Reference, error) {
	head, err := repo.Repository().Head()
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return nil, err
	}
	return head, nil
}

type scopeProvider interface {
	Scope() *gitrepo.Scope
}

// effectiveScope returns the scope a command should operate on.
//
// An explicit non-root override (typically the per-command positional
// argument) wins; otherwise the scope is derived from --repo (or
// $PWD), preserving the legacy "infer scope from path" behaviour.
func effectiveScope(root *cli, repo *gitrepo.Context, overrider scopeProvider) (gitrepo.Scope, error) {
	if s := overrider.Scope(); s != nil {
		return *s, nil
	}

	repoPath := root.Repo
	if repoPath == "" {
		var err error
		if repoPath, err = os.Getwd(); err != nil {
			return gitrepo.Scope{}, err
		}
	}
	return scopeForRepoPath(repo, repoPath)
}
