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
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/0x5a17ed/xit"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type CommitTag struct {
	// TagName of the tag.
	TagName string

	// IsAnnotated describes whenever the Tag references at a Tag Object or a Commit directly.
	// `git describe` prefers annotated tags over lightweight tags.
	IsAnnotated bool

	// CommitHash of the commit the tag points to.
	CommitHash plumbing.Hash

	// TagDate of the tag.
	// - For annotated tags: tagger time.
	// - For lightweight tags: target commit committer time (best practical approximation).
	TagDate time.Time
}

type peeledTag struct {
	commitHash plumbing.Hash
	tagDate    time.Time
}

// peelTagObjectToCommit recursively peels a tag object to a commit object.
func peelTagObjectToCommit(repo *git.Repository, tagObj *object.Tag) (*peeledTag, error) {
	// Usually annotated tags target commits directly.
	if commit, err := repo.CommitObject(tagObj.Target); err == nil {
		return &peeledTag{commit.Hash, commit.Committer.When}, nil
	}

	// Some tags can point to tag objects; peel recursively.
	nextTag, err := repo.TagObject(tagObj.Target)
	switch {
	case errors.Is(err, plumbing.ErrObjectNotFound):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("get tag object: %w", err)
	}

	return peelTagObjectToCommit(repo, nextTag)
}

func peelCommitTagAnnotated(repo *git.Repository, tagObj *object.Tag) (*peeledTag, error) {
	// Peel the tag target to a commit.
	rt, err := peelTagObjectToCommit(repo, tagObj)
	if err != nil {
		return nil, fmt.Errorf("peel tag target to commit: %w", err)
	}

	// Check if the tag target resolved to a commit.
	if rt == nil {
		return nil, nil
	}

	// Annotated tags can have a tagger time.
	if !tagObj.Tagger.When.IsZero() {
		// Override the committer time with the tagger time.
		rt.tagDate = tagObj.Tagger.When
	}

	return rt, nil
}

func peelCommitTagLightweight(repo *git.Repository, ref *plumbing.Reference) (*peeledTag, error) {
	commit, err := repo.CommitObject(ref.Hash())
	switch {
	case errors.Is(err, plumbing.ErrObjectNotFound):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("get commit object: %w", err)
	}

	return &peeledTag{commit.Hash, commit.Committer.When}, nil
}

func resolveCommitTag(repo *git.Repository, ref *plumbing.Reference) (*CommitTag, error) {
	var annotated bool
	var tc *peeledTag

	// First, try resolving hash to an annotated tag object.
	switch tagObj, err := repo.TagObject(ref.Hash()); {
	case errors.Is(err, plumbing.ErrObjectNotFound):
		// Looks like a lightweight tag without annotations.
		annotated = false

		tc, err = peelCommitTagLightweight(repo, ref)
		if err != nil {
			return nil, fmt.Errorf("resolve tag lightweight: %w", err)
		}

	case err != nil:
		// Unexpected error.
		return nil, fmt.Errorf("get tag object: %w", err)

	default:
		// This is an annotated tag.
		annotated = true

		tc, err = peelCommitTagAnnotated(repo, tagObj)
		if err != nil {
			return nil, fmt.Errorf("resolve tag annotated: %w", err)
		}
	}

	// Only keep the tag if it points to a commit.
	if tc == nil {
		return nil, nil
	}

	rvt := &CommitTag{
		TagName:     ref.Name().Short(),
		IsAnnotated: annotated,
		CommitHash:  tc.commitHash,
		TagDate:     tc.tagDate,
	}

	return rvt, nil
}

// IterCommitTags returns an iterator over all tags resolving to a commit in the repository.
func IterCommitTags(gCx *Context) (iter.Seq[CommitTag], func() error) {
	return xit.Perform(func(yield func(CommitTag) bool) error {
		r := gCx.Repository()

		tagIter, err := r.Tags()
		if err != nil {
			return fmt.Errorf("list tags: %w", err)
		}

		walkerFn := func(ref *plumbing.Reference) error {
			switch tag, err := resolveCommitTag(r, ref); {
			case err != nil:
				return fmt.Errorf("resolve tag %q: %w", ref.Name().Short(), err)
			case tag != nil:
				yield(*tag)
			}

			return nil
		}

		if err := tagIter.ForEach(walkerFn); err != nil {
			return fmt.Errorf("iterate tags: %w", err)
		}

		return nil
	})
}

// VersionTag represents a tagged commit with a tag name formatted as a version.
type VersionTag struct {
	CommitTag

	// VersionSpec is the extracted version from the tag.
	VersionSpec VersionSpec
}

func (c VersionTag) String() string {
	return fmt.Sprintf("%s (%s)", c.TagName, c.CommitHash)
}

// FilterMapVersionTags returns an iterator that yields VersionTag objects.
func FilterMapVersionTags(seq iter.Seq[CommitTag]) iter.Seq[VersionTag] {
	return xit.FilterMap2(seq, func(t CommitTag) (VersionTag, bool) {
		vs, err := ParseVersionSpec(t.TagName)
		if err != nil {
			return VersionTag{}, false
		}

		vt := VersionTag{
			CommitTag:   t,
			VersionSpec: vs,
		}

		return vt, true
	})
}

// IterVersionTags returns an iterator over all version tags in the repository.
func IterVersionTags(gCx *Context, scope *Scope) (iter.Seq[VersionTag], func() error) {
	// Iterate over all tags resolving to a commit.
	taggedCommits, doneFn := IterCommitTags(gCx)

	// Map tagged commits to version tags.
	versionTags := FilterMapVersionTags(taggedCommits)

	// Filter version tags by scope if given.
	if scope != nil {
		versionTags = FilterVersionScope(versionTags, *scope)
	}

	return versionTags, doneFn
}

// FilterVersionScope returns an iterator that yields only the tags
// whose scope matches the given Scope.
func FilterVersionScope(seq iter.Seq[VersionTag], scope Scope) iter.Seq[VersionTag] {
	return xit.Filter(seq, func(tag VersionTag) bool {
		return scope.Matches(tag.VersionSpec.Scope)
	})
}

// VersionTagMap maps git plumbing.Hash to one or more VersionTag.
type VersionTagMap map[plumbing.Hash][]VersionTag

// CollectVersionTagMap collects all version tags in the given iterator into a map.
func CollectVersionTagMap(seq iter.Seq[VersionTag]) (out VersionTagMap) {
	out = make(VersionTagMap)
	for tag := range seq {
		out[tag.CommitHash] = append(out[tag.CommitHash], tag)
	}

	return out
}

// NewVersionTagMapFromRepo returns a map of git plumbing.Hash pointing to one or
// more annotated and unannotated tag names.
func NewVersionTagMapFromRepo(gCx *Context, scope *Scope) (out VersionTagMap, err error) {
	versionTagsIter, doneFn := IterVersionTags(gCx, scope)

	// Collect version tags into a map.
	out = CollectVersionTagMap(versionTagsIter)

	if err := doneFn(); err != nil {
		return nil, fmt.Errorf("collect tags: %w", err)
	}

	return out, nil
}
