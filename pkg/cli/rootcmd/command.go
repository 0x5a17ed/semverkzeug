/*
 * Copyright(C) 2022 individual contributors
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

package rootcmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/0x5a17ed/semverkzeug/pkg/cli"
	"github.com/0x5a17ed/semverkzeug/pkg/cli/bumpcmd"
	"github.com/0x5a17ed/semverkzeug/pkg/cli/describecmd"
	"github.com/0x5a17ed/semverkzeug/pkg/cli/versioncmd"
	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/version"
)

var (
	repoPath string
)

// scopeForRepoPath resolves p into a tag scope relative to the
// repository worktree root.
//
// It returns the root scope for root-scoped operation, or when no
// worktree is available.
func scopeForRepoPath(repo *git.Repository, p string) (gitrepo.Scope, error) {
	if p == "" {
		return gitrepo.RootScope(), nil
	}

	// Resolve the user input to an absolute path so that the following
	// path math is stable regardless of the current working directory.
	absPath, err := filepath.Abs(p)
	if err != nil {
		return gitrepo.Scope{}, err
	}

	// Discover the repository root from the checked-out worktree.
	// If no worktree is available, fall back to the root scope.
	wt, err := repo.Worktree()
	if err != nil {
		return gitrepo.RootScope(), nil
	}

	// Normalize the worktree root to an absolute path to keep comparison
	// logic consistent with absPath above.
	rootPath, err := filepath.Abs(wt.Filesystem.Root())
	if err != nil {
		return gitrepo.Scope{}, err
	}

	// Convert the target path into a path relative to the repository root.
	// This relative segment becomes the tag scope (for example, a submodule).
	relPath, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		return gitrepo.Scope{}, err
	}

	// Convert separators to "/" so scope values are platform-independent.
	relPath = filepath.ToSlash(relPath)

	return gitrepo.ParseScope(relPath)
}

func persistentPreRunE(cmd *cobra.Command, args []string) (err error) {
	if repoPath == "" {
		var err error
		if repoPath, err = os.Getwd(); err != nil {
			return err
		}
	}

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if errors.Is(err, git.ErrRepositoryNotExists) {
		return nil
	}

	scope, err := scopeForRepoPath(repo, repoPath)
	if err != nil {
		return err
	}

	ctx := cli.WithGitRepository(cmd.Context(), repo)
	ctx = cli.WithScope(ctx, scope)
	cmd.SetContext(ctx)
	return
}

func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "semverkzeug",
		Short: "versioning tool for git repositories",

		PersistentPreRunE: persistentPreRunE,
	}

	if v, err := version.GetVersion(); err == nil {
		c.Flags().Bool("version", false, "version for this command")
		_ = c.Flags().MarkHidden("version")

		c.Version = v
	}

	pfs := c.PersistentFlags()
	pfs.StringVarP(&repoPath, "repo", "r", "", "git repository path (default is $PWD)")

	c.AddCommand(describecmd.Command())
	c.AddCommand(bumpcmd.Command())
	c.AddCommand(versioncmd.Command())

	return c
}

func Execute(args []string) error {
	c := Command()
	c.SetArgs(args)
	return c.Execute()
}
