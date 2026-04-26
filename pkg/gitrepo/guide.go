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

func SelectHighestVersionTag(tags []VersionTaggedCommit) VersionTaggedCommit {
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
		if !a.Date.Equal(b.Date) {
			return a.Date.After(b.Date)
		}

		// Use lexicographic order for tags with the same version as the last tie-breaker.
		return a.TagName > b.TagName
	})

	return cp[0]
}

// Guide describes the distance in Depth of the value in Hash to
// the commit with the given Tags or the initial commit if there
// are no tags.
type Guide struct {
	// Commit is the commit at the given Hash.
	Commit *object.Commit

	// Tags is the list of tags that point to the given commit.
	Tags []VersionTaggedCommit

	// Depth is the distance in commits from the given Hash to the commit with the given Tags.
	Depth int

	r *git.Repository
}

func (g Guide) HasCommit() bool {
	return g.Commit != nil && !g.Commit.Hash.IsZero()
}

// NewGuide looks at the given git repository and gives the given
// plumbing.Reference ref a human-readable name based on another
// available ref.
func NewGuide(repo *git.Repository, ref *plumbing.Reference, scope Scope) (*Guide, error) {
	guide := &Guide{r: repo}

	if ref == nil {
		return guide, nil
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("resolve commit object: %w", err)
	}
	guide.Commit = commit

	tm, err := NewVersionTagMapFromRepo(repo, &scope)
	if err != nil {
		return nil, fmt.Errorf("build version tag map: %w", err)
	}

	// Follow the parents until we find a version tag.
	for c := guide.Commit; c != nil; {
		if ts, ok := tm[c.Hash]; ok && len(ts) > 0 {
			guide.Tags = ts
			return guide, nil
		}

		guide.Depth++

		if c.NumParents() == 0 {
			// No parents left, so we're at the root'.
			break
		}

		// Follow the first parent.
		if c, err = c.Parent(0); err != nil {
			return nil, fmt.Errorf("read first parent: %w", err)
		}
	}

	// No version tag was found.
	return guide, nil
}
