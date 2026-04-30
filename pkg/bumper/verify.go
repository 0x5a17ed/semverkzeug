package bumper

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

// VerifyRepo validates the repository state by checking the reference and
// ensuring a clean working tree status.
func VerifyRepo(cx *gitrepo.Context, ref *plumbing.Reference) error {
	if ref == nil {
		return ErrRepositoryIsEmpty
	}

	st, err := gitrepo.BuildWorktreeStatus(cx)
	if err != nil {
		return fmt.Errorf("read worktree status: %w", err)
	}

	if !st.IsClean() {
		return ErrRepositoryIsDirty
	}

	return nil
}
