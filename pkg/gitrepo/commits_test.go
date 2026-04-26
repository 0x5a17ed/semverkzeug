package gitrepo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/testhelper"
)

func TestAbbreviatedCommitHash(t *testing.T) {
	repo := testhelper.RepoEmpty(t)

	h := testhelper.CommitFile(t, repo, "test.txt", "test")

	got, err := AbbreviatedCommitHash(repo.Repo, h)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s", h.String()[:8]), got)
}
