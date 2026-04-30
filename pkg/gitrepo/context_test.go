package gitrepo

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func must[T any](t *testing.T) func(v T, err error) T {
	return func(v T, err error) T {
		require.NoError(t, err)
		return v
	}
}

func TestContext_String(t *testing.T) {
	type args struct {
		repo *git.Repository

		equalFn func(t *testing.T, b string)
	}
	tests := []struct {
		name string
		args args
	}{
		{"zero value", args{
			repo: nil,
			equalFn: func(t *testing.T, b string) {
				assert.Equal(t, "gitrepo.Context{repo=<nil>}", b)
			},
		}},
		{"memory", args{
			repo: func() *git.Repository {
				return must[*git.Repository](t)(
					git.Init(memory.NewStorage(), nil),
				)
			}(),
			equalFn: func(t *testing.T, b string) {
				assert.Equal(t, "gitrepo.Context{repo=<memory>}", b)
			},
		}},
		{"filesystem", args{
			repo: func() *git.Repository {
				return must[*git.Repository](t)(
					git.PlainInit(t.TempDir(), false),
				)
			}(),
			equalFn: func(t *testing.T, b string) {
				assert.Regexp(t, `gitrepo.Context{repo=<filesystem:path=.*}`, b)
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cx, err := NewContextFromRepo(tt.args.repo)
			require.NoError(t, err)

			tt.args.equalFn(t, cx.String())
		})
	}
}
