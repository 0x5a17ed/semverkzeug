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
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/bumper"
	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

// bumpParts maps the user-facing part name to the bumper.Part value.
var bumpParts = map[string]bumper.Part{
	"major": bumper.Major,
	"minor": bumper.Minor,
	"patch": bumper.Patch,
}

type bumpCmd struct {
	Part     string         `arg:"" enum:"major,minor,patch" help:"part of the version to bump (major, minor, patch)"`
	ScopeArg *gitrepo.Scope `arg:"true" name:"scope" optional:"" help:"tag scope to bump (defaults to scope derived from --repo)"`
}

func (c *bumpCmd) Scope() *gitrepo.Scope { return c.ScopeArg }

func (c *bumpCmd) Run(root *cli, repo *gitrepo.Context, head *plumbing.Reference) error {
	scope, err := effectiveScope(root, repo, c)
	if err != nil {
		return err
	}

	part := bumpParts[c.Part]

	if _, err := bumper.CreateTag(repo, head, part, scope); err != nil {
		return err
	}
	return nil
}
