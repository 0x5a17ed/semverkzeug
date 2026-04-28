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
	"fmt"
	"slices"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Guide describes the relationship between a reference (typically
// HEAD) and the highest version tag it shares history with.
type Guide struct {
	// Commit is the commit at the given ref.
	Commit *object.Commit

	// Tags is the set of version tags that share the highest
	// reachable version (one entry in the common case, more if
	// several tags name the same version).
	Tags []VersionTag

	// MergeBase is the most recent commit shared between Commit and
	// the winning tag's commit.  Equal to Commit when the tag is on
	// Commit itself, or to the tag's commit on linear ancestry; the
	// branch point for tags that live on side branches.  Nil when no
	// reachable tag was found.
	MergeBase *object.Commit

	// Depth is the number of commits reachable from Commit but not
	// from MergeBase — equivalent to `git rev-list --count
	// MergeBase..Commit`.  Zero when Commit is at MergeBase.  When
	// no tag was found, Depth is the total number of commits
	// reachable from Commit.
	Depth int
}

func (g Guide) String() string {
	commitString := "nil"
	if g.HasCommit() {
		commitString = fmt.Sprintf("(%s)", g.Commit.Hash)
	}

	mergeBaseString := "nil"
	if g.MergeBase != nil {
		mergeBaseString = fmt.Sprintf("(%s)", g.MergeBase.Hash)
	}

	return fmt.Sprintf("Guide{Commit: %s, MergeBase: %s, Depth: %d}", commitString, mergeBaseString, g.Depth)
}

// HasCommit checks if the Guide has a valid non-nil commit with a non-zero hash.
func (g Guide) HasCommit() bool {
	return g.Commit != nil && !g.Commit.Hash.IsZero()
}

// NewGuide finds the highest version tag whose commit shares
// history with ref, then describes how far ref has diverged from
// that tag's branch point.
//
// Tags whose commit is not reachable from any branch (local or
// remote-tracking) are skipped to avoid picking up stranded or
// experimental work.  Tags that sit "in the future" of ref (ref is
// a strict ancestor of the tag) are also skipped — they describe
// versions that don't exist yet from ref's perspective.
func NewGuide(gCx *Context, ref *plumbing.Reference, scope Scope) (*Guide, error) {
	r := gCx.Repository()

	if ref == nil {
		return &Guide{}, nil
	}

	head, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("resolve commit object: %w", err)
	}

	tm, err := NewVersionTagMapFromRepo(gCx, &scope)
	if err != nil {
		return nil, fmt.Errorf("build version tag map: %w", err)
	}

	if len(tm) > 0 {
		guide, err := selectReachableTag(r, head, tm)
		if err != nil {
			return nil, fmt.Errorf("select reachable tag: %w", err)
		}
		if guide != nil {
			return guide, nil
		}
	}

	// No reachable version tag.  Depth becomes "everything reachable
	// from ref" so callers can tell whether any history exists.
	depth, err := countReachable(head)
	if err != nil {
		return nil, fmt.Errorf("count reachable commits: %w", err)
	}

	guide := &Guide{
		Commit: head,
		Depth:  depth,
	}

	return guide, nil
}

// SelectHighestVersionTag returns the highest version tag from a list of
// VersionTag based on version and metadata hierarchy.
func SelectHighestVersionTag(tags []VersionTag) VersionTag {
	cp := slices.Clone(tags)

	sort.Slice(cp, func(i, j int) bool {
		a, b := cp[i], cp[j]

		// Sort highest version first.
		if a.VersionSpec.Version.GreaterThan(&b.VersionSpec.Version) {
			return true
		}
		if a.VersionSpec.Version.LessThan(&b.VersionSpec.Version) {
			return false
		}

		// Prefer annotated tags over lightweight tags.
		if a.IsAnnotated != b.IsAnnotated {
			return a.IsAnnotated && !b.IsAnnotated
		}

		// Prefer tags with a later date.
		if !a.TagDate.Equal(b.TagDate) {
			return a.TagDate.After(b.TagDate)
		}

		// Use lexicographic order for tags with the same version as the last tie-breaker.
		return a.TagName > b.TagName
	})

	return cp[0]
}

// selectReachableTag picks the highest-semver tag from tm whose
// commit shares non-trivial history with head, applying the
// tip-filter and the "future tag" guard.  Returns nil when no tag
// qualifies (caller should fall back to the no-tag path).
func selectReachableTag(
	repo *git.Repository,
	head *object.Commit,
	tm VersionTagMap,
) (*Guide, error) {
	tipSet, err := buildTipSet(repo)
	if err != nil {
		return nil, fmt.Errorf("build tip set: %w", err)
	}

	candidates := flattenTagMap(tm)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].VersionSpec.Version.GreaterThan(&candidates[j].VersionSpec.Version)
	})

	for _, vtc := range candidates {
		// Skip tags on stranded commits when we have a tip-set to
		// compare against.  An empty tip-set means the repo has no
		// branches at all; fall back to considering every tag in
		// that case.
		if len(tipSet) > 0 && !tipSet[vtc.CommitHash] {
			continue
		}

		tagCommit, err := repo.CommitObject(vtc.CommitHash)
		if err != nil {
			return nil, fmt.Errorf("resolve tag commit %s: %w", vtc.CommitHash, err)
		}

		bases, err := head.MergeBase(tagCommit)
		if err != nil {
			return nil, fmt.Errorf("compute merge-base: %w", err)
		}
		if len(bases) == 0 {
			continue
		}
		mergeBase := bases[0]

		// Skip tags from head's "future": head is a strict ancestor
		// of the tag, not at it.
		if mergeBase.Hash == head.Hash && head.Hash != vtc.CommitHash {
			continue
		}

		depth, err := commitsAhead(head, mergeBase)
		if err != nil {
			return nil, fmt.Errorf("count divergence: %w", err)
		}

		return &Guide{
			Commit:    head,
			Tags:      collectSameVersion(candidates, vtc.VersionSpec),
			MergeBase: mergeBase,
			Depth:     depth,
		}, nil
	}

	return nil, nil
}

func flattenTagMap(tm VersionTagMap) []VersionTag {
	out := make([]VersionTag, 0, len(tm))
	for _, tags := range tm {
		out = append(out, tags...)
	}
	return out
}

func collectSameVersion(candidates []VersionTag, spec VersionSpec) []VersionTag {
	var out []VersionTag
	for _, c := range candidates {
		if c.VersionSpec.Version.Equal(&spec.Version) {
			out = append(out, c)
		}
	}
	return out
}

// buildTipSet returns the set of commit hashes reachable from any
// local branch or remote-tracking ref.  Remote-tracking refs are
// included because typical CI checkouts and fresh clones leave
// every non-checked-out branch as `refs/remotes/origin/*`; ignoring
// them would wrongly classify legitimate release-branch tags as
// stranded.  An empty set means the repo has no branches at all,
// in which case callers should treat every tag as in-scope.
func buildTipSet(repo *git.Repository) (map[plumbing.Hash]bool, error) {
	set := map[plumbing.Hash]bool{}

	refIter, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("list references: %w", err)
	}

	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name()
		if !name.IsBranch() && !name.IsRemote() {
			return nil
		}
		commit, cErr := repo.CommitObject(ref.Hash())
		if cErr != nil {
			// Refs that don't resolve to a commit (broken or
			// symbolic refs) don't contribute to reachability.
			return nil
		}
		iter := object.NewCommitPreorderIter(commit, set, nil)
		return iter.ForEach(func(c *object.Commit) error {
			set[c.Hash] = true
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("walk references: %w", err)
	}

	return set, nil
}

// commitsAhead counts commits reachable from head but not from
// mergeBase. Same as `git rev-list --count mergeBase..head`.
func commitsAhead(head, mergeBase *object.Commit) (int, error) {
	if head.Hash == mergeBase.Hash {
		return 0, nil
	}

	excluded := map[plumbing.Hash]bool{}
	mergeBaseIter := object.NewCommitPreorderIter(mergeBase, nil, nil)
	if err := mergeBaseIter.ForEach(func(c *object.Commit) error {
		excluded[c.Hash] = true
		return nil
	}); err != nil {
		return 0, fmt.Errorf("walk merge-base ancestors: %w", err)
	}

	count := 0
	headIter := object.NewCommitPreorderIter(head, excluded, nil)
	if err := headIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	}); err != nil {
		return 0, fmt.Errorf("walk head ancestors: %w", err)
	}
	return count, nil
}

// countReachable counts every commit reachable from the head, including
// the head itself.
func countReachable(head *object.Commit) (int, error) {
	count := 0
	iter := object.NewCommitPreorderIter(head, nil, nil)
	err := iter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	return count, err
}
