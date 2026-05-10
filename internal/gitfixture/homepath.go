package gitfixture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type HomeFixture func(parts ...string) string

func NewHomeFixture(t *testing.T) HomeFixture {
	t.Helper()

	path := t.TempDir()

	t.Setenv("HOME", path)
	return func(parts ...string) string {
		return filepath.Join(append([]string{path}, parts...)...)
	}
}

func WriteFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
