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
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type hashPrefixStorer interface {
	HashesWithPrefix(prefix []byte) ([]plumbing.Hash, error)
}

// minAbbrevNibbles is the floor length, in hex characters, for an abbreviated
// commit hash. Matches git's default core.abbrev.
const minAbbrevNibbles = 7

func commonPrefixLength(a, b string) (i int) {
	n := min(len(a), len(b))
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func abbreviateCommitWithHashPrefix(r *git.Repository, h plumbing.Hash, st hashPrefixStorer) (string, error) {
	// HashesWithPrefix only accepts byte-aligned prefixes, but we want
	// nibble-aligned answers down to minAbbrevNibbles. A single query at the
	// floor byte boundary returns a superset of every commit that could
	// collide at any k >= minAbbrevNibbles, so one round trip is enough.
	byteFloor := minAbbrevNibbles / 2
	hashes, err := st.HashesWithPrefix(h[:byteFloor])
	if err != nil {
		return "", fmt.Errorf("find hashes with prefix %x: %w", h[:byteFloor], err)
	}

	full := h.String()
	maxCollision := 0
	for _, candidate := range hashes {
		if candidate == h {
			continue
		}
		switch _, err := r.CommitObject(candidate); {
		case errors.Is(err, plumbing.ErrObjectNotFound):
			continue
		case err != nil:
			return "", fmt.Errorf("resolve candidate commit %s: %w", candidate, err)
		}

		if lcp := commonPrefixLength(full, candidate.String()); lcp > maxCollision {
			maxCollision = lcp
		}
	}

	k := max(maxCollision+1, minAbbrevNibbles)
	if k > len(full) {
		return full, nil
	}
	return full[:k], nil
}

func abbreviateCommitByScanning(r *git.Repository, h plumbing.Hash) (string, error) {
	full := h.String()

	for i, n := minAbbrevNibbles, len(full); i <= n; i++ {
		prefix := full[:i]

		iter, err := r.CommitObjects()
		if err != nil {
			return "", fmt.Errorf("iterate commit objects: %w", err)
		}

		commitMatches := 0
		matchedTarget := false

		err = iter.ForEach(func(c *object.Commit) error {
			if !strings.HasPrefix(c.Hash.String(), prefix) {
				return nil
			}

			commitMatches++

			if c.Hash == h {
				matchedTarget = true
			}

			if commitMatches > 1 {
				return storer.ErrStop
			}

			return nil
		})

		if err != nil && !errors.Is(err, storer.ErrStop) {
			return "", fmt.Errorf("scan commit objects for prefix %q: %w", prefix, err)
		}

		if matchedTarget && commitMatches == 1 {
			return full[:i], nil
		}
	}

	return full, nil
}

// FindUniqueCommitHashAbbreviation returns a shortened hash of the commit that uniquely identifies the commit.
func FindUniqueCommitHashAbbreviation(cx *Context, co *object.Commit) (string, error) {
	if co == nil || co.Hash == plumbing.ZeroHash {
		return "", fmt.Errorf("commit is nil or has zero hash")
	}

	r := cx.Repository()

	if store, ok := r.Storer.(hashPrefixStorer); ok {
		return abbreviateCommitWithHashPrefix(r, co.Hash, store)
	}

	return abbreviateCommitByScanning(r, co.Hash)
}
