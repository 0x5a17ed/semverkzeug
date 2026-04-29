package gitrepo

import (
	"encoding/hex"
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

func abbreviateCommitWithHashPrefix(r *git.Repository, h plumbing.Hash, st hashPrefixStorer) (string, error) {
	for i, n := 4, len(h); i <= n; i++ {
		prefix := h[:i]

		hashes, err := st.HashesWithPrefix(prefix)
		if err != nil {
			return "", fmt.Errorf("find hashes with prefix %q: %w", prefix, err)
		}

		commitMatches := 0
		matchedTarget := false
		for _, candidate := range hashes {
			switch _, err := r.CommitObject(candidate); {
			case errors.Is(err, plumbing.ErrObjectNotFound):
				continue
			case err != nil:
				return "", fmt.Errorf("resolve candidate commit %s: %w", candidate, err)
			}

			commitMatches++
			if candidate == h {
				matchedTarget = true
			}
			if commitMatches > 1 {
				break
			}
		}

		if matchedTarget && commitMatches == 1 {
			return hex.EncodeToString(h[:i]), nil
		}
	}

	return h.String(), nil
}

func abbreviateCommitByScanning(r *git.Repository, h plumbing.Hash) (string, error) {
	full := h.String()

	for i, n := 7, len(full); i <= n; i++ {
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
