package bumper_test

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0x5a17ed/semverkzeug/pkg/bumper"
	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/internal/gitfixture"
)

func TestCreateTag_FilesystemRepositoryHappyPath(t *testing.T) {
	// Arrange
	cx := gitfixture.RepoWithOneCommitOneTagClean(t)

	t.Setenv("GIT_AUTHOR_NAME", gitfixture.TestSig.Name)
	t.Setenv("GIT_AUTHOR_EMAIL", gitfixture.TestSig.Email)
	t.Setenv("GIT_COMMITTER_NAME", gitfixture.TestSig.Name)
	t.Setenv("GIT_COMMITTER_EMAIL", gitfixture.TestSig.Email)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), ".gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "tag.gpgSign")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")

	head := gitfixture.Head(t, cx)

	// Act
	tagRef, err := bumper.CreateTag(cx, head, bumper.Patch, gitrepo.RootScope())

	// Assert
	require.NoError(t, err)

	assert.Equal(t, plumbing.NewTagReferenceName("v0.1.1"), tagRef.Name())

	resolvedRef, err := cx.Repository().Tag("v0.1.1")
	require.NoError(t, err)
	assert.Equal(t, tagRef.Hash(), resolvedRef.Hash())

	tagObject, err := cx.Repository().TagObject(tagRef.Hash())
	require.NoError(t, err)
	assert.Equal(t, "v0.1.1", tagObject.Name)
	assert.Equal(t, "bump version v0.1.0 -> v0.1.1\n", tagObject.Message)
}
