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
	"bytes"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	fsnode "github.com/go-git/go-git/v5/utils/merkletrie/filesystem"
	indexnode "github.com/go-git/go-git/v5/utils/merkletrie/index"
	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
)

// BuildWorktreeStatus computes the worktree status of the current repository
// context by comparing the working tree state with the index using Git-compatible
// ignore precedence for untracked files.
// Returns a [git.Status] or an error.
func BuildWorktreeStatus(cx *Context) (git.Status, error) {
	matcher, err := loadIgnoreMatcher(cx)
	if err != nil {
		return nil, fmt.Errorf("load gitignore matcher: %w", err)
	}

	head := plumbing.ZeroHash
	ref, err := cx.Repository().Head()
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return nil, fmt.Errorf("resolve HEAD: %w", err)
	}
	if err == nil {
		head = ref.Hash()
	}

	return status(cx, head, matcher)
}

func status(cx *Context, commit plumbing.Hash, matcher *ignoreMatcher) (git.Status, error) {
	st := make(git.Status)

	left, err := diffCommitWithStaging(cx.Repository(), commit, false)
	if err != nil {
		return nil, err
	}

	for _, ch := range left {
		action, err := ch.Action()
		if err != nil {
			return nil, err
		}

		fs := st.File(nameFromAction(&ch))
		fs.Worktree = git.Unmodified

		switch action {
		case merkletrie.Delete:
			st.File(ch.From.String()).Staging = git.Deleted
		case merkletrie.Insert:
			st.File(ch.To.String()).Staging = git.Added
		case merkletrie.Modify:
			st.File(ch.To.String()).Staging = git.Modified
		}
	}

	right, err := diffStagingWithWorktree(cx, false)
	if err != nil {
		return nil, err
	}

	for _, ch := range right {
		action, err := ch.Action()
		if err != nil {
			return nil, err
		}

		if action == merkletrie.Insert && matcher.Match(nameFromAction(&ch), changeIsDir(&ch)) {
			continue
		}

		fs := st.File(nameFromAction(&ch))
		if fs.Staging == git.Untracked {
			fs.Staging = git.Unmodified
		}

		switch action {
		case merkletrie.Delete:
			fs.Worktree = git.Deleted
		case merkletrie.Insert:
			fs.Worktree = git.Untracked
			fs.Staging = git.Untracked
		case merkletrie.Modify:
			fs.Worktree = git.Modified
		}
	}

	return st, nil
}

func diffCommitWithStaging(repo *git.Repository, commit plumbing.Hash, reverse bool) (merkletrie.Changes, error) {
	var tree *object.Tree
	if !commit.IsZero() {
		c, err := repo.CommitObject(commit)
		if err != nil {
			return nil, err
		}

		tree, err = c.Tree()
		if err != nil {
			return nil, err
		}
	}

	return diffTreeWithStaging(repo, tree, reverse)
}

func diffTreeWithStaging(repo *git.Repository, tree *object.Tree, reverse bool) (merkletrie.Changes, error) {
	var from noder.Noder
	if tree != nil {
		from = object.NewTreeRootNode(tree)
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return nil, err
	}

	to := indexnode.NewRootNode(idx)
	if reverse {
		return merkletrie.DiffTree(to, from, diffTreeIsEqual)
	}

	return merkletrie.DiffTree(from, to, diffTreeIsEqual)
}

func diffStagingWithWorktree(cx *Context, reverse bool) (merkletrie.Changes, error) {
	idx, err := cx.Repository().Storer.Index()
	if err != nil {
		return nil, err
	}

	wt, err := cx.LoadWorktree()
	if err != nil {
		return nil, err
	}

	from := indexnode.NewRootNode(idx)

	submodules, err := submoduleStatus(wt)
	if err != nil {
		return nil, err
	}
	to := fsnode.NewRootNodeWithOptions(wt.Filesystem, submodules, fsnode.Options{Index: idx})

	if reverse {
		return merkletrie.DiffTree(to, from, diffTreeIsEqual)
	}

	return merkletrie.DiffTree(from, to, diffTreeIsEqual)
}

func submoduleStatus(wt *git.Worktree) (map[string]plumbing.Hash, error) {
	out := map[string]plumbing.Hash{}

	submodules, err := wt.Submodules()
	if err != nil {
		return nil, err
	}

	status, err := submodules.Status()
	if err != nil {
		return nil, err
	}

	for _, s := range status {
		if s.Current.IsZero() {
			out[s.Path] = s.Expected
			continue
		}

		out[s.Path] = s.Current
	}

	return out, nil
}

func nameFromAction(ch *merkletrie.Change) string {
	name := ch.To.String()
	if name == "" {
		return ch.From.String()
	}

	return name
}

func changeIsDir(ch *merkletrie.Change) bool {
	return (len(ch.To) > 0 && ch.To.IsDir()) || (len(ch.From) > 0 && ch.From.IsDir())
}

var emptyNoderHash = make([]byte, 24)

func diffTreeIsEqual(a, b noder.Hasher) bool {
	hashA := a.Hash()
	hashB := b.Hash()

	if bytes.Equal(hashA, emptyNoderHash) || bytes.Equal(hashB, emptyNoderHash) {
		return false
	}

	return bytes.Equal(hashA, hashB)
}
