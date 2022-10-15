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

package bumpcmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"

	"github.com/0x5a17ed/semverkzeug/pkg/bump"
	"github.com/0x5a17ed/semverkzeug/pkg/cli"
)

var (
	partMap = map[string]bump.Part{
		"major": bump.Major,
		"minor": bump.Minor,
		"patch": bump.Patch,
	}

	partKeys = (func() (out []string) {
		for k, _ := range partMap {
			out = append(out, k)
		}
		return
	})()
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

	newTag, err := bump.It(repo, head, partMap[args[0]])
	if err != nil {
		return err
	}

	fmt.Printf("created new tag: %s\n", newTag.Name().Short())

	return nil
}

func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "bump [major|minor|patch]",
		Short: "Bumps the current version and creates a new tag",
		Long: `Prints a version string describing the 
currently checked out revision.`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: partKeys,
		Run:       cli.RunCatchErr(runE),
	}

	return c
}
