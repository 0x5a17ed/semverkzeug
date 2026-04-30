package gitrepo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScope(t *testing.T) {
	type args struct {
		inp     string
		want    string
		wantErr bool
	}

	tt := []struct {
		name string
		args args
	}{
		{"empty string is root", args{
			inp:  "",
			want: "",
		}},
		{"dot is root", args{
			inp:  ".",
			want: "",
		}},
		{"slash is root", args{
			inp:  "/",
			want: "",
		}},
		{"whitespace is trimmed", args{
			inp:  "  mod/sub  ",
			want: "mod/sub",
		}},
		{"single segment", args{
			inp:  "mod",
			want: "mod",
		}},
		{"nested scope", args{
			inp:  "pkg/sub.mod-1",
			want: "pkg/sub.mod-1",
		}},
		{"duplicate separators are collapsed", args{
			inp:  "pkg//sub///mod",
			want: "pkg/sub/mod",
		}},
		{"current directory segments are collapsed", args{
			inp:  "pkg/./sub",
			want: "pkg/sub",
		}},
		{"non escaping parent segments are collapsed", args{
			inp:  "pkg/../sub",
			want: "sub",
		}},
		{"absolute non root path is rejected", args{
			inp:     "/pkg/sub",
			wantErr: true,
		}},
		{"parent directory escape is rejected", args{
			inp:     "../pkg",
			wantErr: true,
		}},
		{"parent directory escape after clean is rejected", args{
			inp:     "pkg/../../sub",
			wantErr: true,
		}},
		{"segment must start with alphanumeric", args{
			inp:     "pkg/_sub",
			wantErr: true,
		}},
		{"invalid character is rejected", args{
			inp:     "pkg/sub~mod",
			wantErr: true,
		}},
		{"windows separators are rejected", args{
			inp:     `pkg\sub`,
			wantErr: true,
		}},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			raw := tc.args.inp

			// Act
			got, err := ParseScope(raw)

			// Assert
			if tc.args.wantErr {
				assert.Error(t, err)
				assert.Zero(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.args.want, got.String())
			assert.Equal(t, tc.args.want == "", got.IsRoot())
		})
	}
}
