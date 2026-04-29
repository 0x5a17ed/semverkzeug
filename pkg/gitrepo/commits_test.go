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

	c := gitfixture.CommitFile(t, repo, "test.txt", "test")

	got, err := gitrepo.FindUniqueCommitHashAbbreviation(repo, c)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s", c.Hash.String()[:8]), got)
}
