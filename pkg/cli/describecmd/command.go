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
	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
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

	scope, _ := cli.GetScope(ctx)
	vs, err := floatingversion.Describe(repo, head, scope)
	if err != nil {
		return err
	}

	// Add the commit hash to the version if requested.
	if withCommitHash && vs.Guide.HasCommit() {
		abbreviatedHash, err := gitrepo.AbbreviatedCommitHash(repo, vs.Guide.Commit.Hash)
		if err != nil {
			return fmt.Errorf("abbreviate commit hash: %w", err)
		}

		vs.Spec.Version, err = vs.Spec.Version.SetMetadata("g" + abbreviatedHash)
		if err != nil {
			return fmt.Errorf("set metadata: %w", err)
		}
	}

	if noPrefix {
		_, err = fmt.Println(vs.Spec.Version.String())
	} else {
		_, err = fmt.Println(vs.Spec.String())
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
