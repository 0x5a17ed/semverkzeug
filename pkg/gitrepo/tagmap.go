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
)

type TagMap map[plumbing.Hash][]string

func (m TagMap) FindMatches(h *plumbing.Hash, p *regexp.Regexp) []string {
	for _, tag := range m[*h] {
		if p == nil || p.MatchString(tag) {
			return m[*h]
		}
	}
	return nil
}

// GetTagMap returns a map of git plumbing.Hash pointing to one or
// more annotated and unannotated tag names.
func GetTagMap(repo *git.Repository) (out TagMap, err error) {
	tags, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	out = make(TagMap)
	err = tags.ForEach(func(r *plumbing.Reference) error {
		tag, err := repo.TagObject(r.Hash())
		switch err {
		case nil:
			commit, err := tag.Commit()
			if err != nil {
				return nil
			}
			out[commit.Hash] = append(out[commit.Hash], tag.Name)
		case plumbing.ErrObjectNotFound:
			out[r.Hash()] = append(out[r.Hash()], r.Name().Short())
		default:
			return err
		}
		return nil
	})
	return
}
