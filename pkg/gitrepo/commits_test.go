package gitrepo_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/internal/gitfixture"
)

func TestAbbreviatedCommitHash(t *testing.T) {
	repo := gitfixture.RepoEmpty(t)

	h := gitfixture.CommitFile(t, repo, "test.txt", "test")

	got, err := gitrepo.AbbreviatedCommitHash(repo, h)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s", h.String()[:8]), got)
}
