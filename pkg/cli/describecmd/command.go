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

package describecmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"

	"github.com/0x5a17ed/semverkzeug/pkg/cli"
	"github.com/0x5a17ed/semverkzeug/pkg/floatingversion"
)

var (
	noPrefix       bool
	withCommitHash bool
)

func runE(ctx context.Context, cmd *cobra.Command, args []string) error {
	repo, ok := cli.GetGitRepository(ctx)
	if !ok {
		return git.ErrRepositoryNotExists
	}

	head, err := repo.Head()
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return err
	}

	vs, err := floatingversion.Get(repo, head, withCommitHash)
	if err != nil {
		return err
	}

	if noPrefix {
		_, err = fmt.Println(vs.Version.String())
	} else {
		_, err = fmt.Println(vs.String())
	}
	return err
}

func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "describe",
		Short: "Print current version string",
		Long: `Prints a floating version string describing the 
current state of the repository.`,
		Run: cli.RunCatchErr(runE),
	}

	fl := c.Flags()
	fl.BoolVar(&withCommitHash, "add-commit-hash", false, "add commit hash as metadata")
	fl.BoolVar(&noPrefix, "no-prefix", false, "print the version without prefix")

	return c
}

func Execute(ctx context.Context, args []string) error {
	c := Command()
	c.SetArgs(args)
	return c.ExecuteContext(ctx)
}
