/*
 * Copyright(C) 2026 the semverkzeug developers
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

package main

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/floatingversion"
	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

type describeCmd struct {
	ScopeArg *gitrepo.Scope `arg:"true" name:"scope" optional:"" help:"tag scope to describe (defaults to scope derived from --repo)"`

	AddCommitHash bool `name:"add-commit-hash" help:"add commit hash as metadata"`
	NoPrefix      bool `name:"no-prefix" help:"print the version without prefix"`
}

func (c *describeCmd) Scope() *gitrepo.Scope { return c.ScopeArg }

func (c *describeCmd) Run(root *cli, repo *gitrepo.Context, head *plumbing.Reference) error {
	scope, err := effectiveScope(root, repo, c)
	if err != nil {
		return err
	}

	guide, err := gitrepo.BuildGuide(repo, head, scope)
	if err != nil {
		return fmt.Errorf("build guide: %w", err)
	}

	spec, err := floatingversion.Describe(repo, guide)
	if err != nil {
		return err
	}

	// Add the commit hash to the version if requested.
	if c.AddCommitHash && guide.HasCommit() {
		abbreviatedHash, err := gitrepo.FindUniqueCommitHashAbbreviation(repo, guide.Commit)
		if err != nil {
			return fmt.Errorf("abbreviate commit hash: %w", err)
		}

		v, err := spec.Version.SetMetadata("g" + abbreviatedHash)
		if err != nil {
			return fmt.Errorf("set metadata: %w", err)
		}
		spec = spec.WithVersion(v)
	}

	if c.NoPrefix {
		_, err = fmt.Println(spec.Version.String())
	} else {
		_, err = fmt.Println(spec.String())
	}
	return err
}
