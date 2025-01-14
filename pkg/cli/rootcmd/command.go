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

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"

	"github.com/0x5a17ed/semverkzeug/pkg/cli"
	"github.com/0x5a17ed/semverkzeug/pkg/cli/bumpcmd"
	"github.com/0x5a17ed/semverkzeug/pkg/cli/describecmd"
	"github.com/0x5a17ed/semverkzeug/pkg/cli/versioncmd"
	"github.com/0x5a17ed/semverkzeug/pkg/version"
)

var (
	repoPath string
)

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

	cmd.SetContext(cli.WithGitRepository(cmd.Context(), repo))
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
