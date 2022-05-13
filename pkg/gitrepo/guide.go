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
	"regexp"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Guide describes the distance in Depth of the value in Hash to
// the commit with the given Tags or the initial commit if there
// are no tags.
type Guide struct {
	Tags  []string
	Depth int
	Hash  plumbing.Hash

	r *git.Repository
}

func (n Guide) AbbreviatedHash() string {
	hString := n.Hash.String()
	hLength := len(hString)
	for i := 7; i < hLength; i++ {
		v := hString[:i]
		h, err := n.r.ResolveRevision(plumbing.Revision(v))
		if err == nil && *h == n.Hash {
			return v
		}
	}
	return hString
}

// GetGuide looks at the given git repository and gives the given
// plumbing.Reference ref a human-readable name based on another
// available ref.
func GetGuide(repo *git.Repository, ref *plumbing.Reference, tagPattern *regexp.Regexp) (guide Guide, err error) {
	guide.r, guide.Hash = repo, ref.Hash()

	tm, err := GetTagMap(repo)
	if err != nil {
		return guide, err
	}

	// Fast path: check if we have a direct hit.
	if ts := tm.FindMatches(&guide.Hash, tagPattern); len(ts) != 0 {
		guide.Tags = ts
		return guide, nil
	}

	commits, err := repo.Log(&git.LogOptions{
		From:  guide.Hash,
		Order: git.LogOrderCommitterTime, // This might need to be LogOrderBSF.
	})
	if err != nil {
		return guide, err
	}
	_ = commits.ForEach(func(c *object.Commit) error {
		if ts := tm.FindMatches(&c.Hash, tagPattern); len(ts) != 0 {
			guide.Tags = ts
			return storer.ErrStop
		}
		guide.Depth += 1
		return nil
	})
	return guide, nil
}
