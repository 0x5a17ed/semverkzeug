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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/internal/bumper"
	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
	"github.com/0x5a17ed/semverkzeug/internal/uiprint"
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

	MessageFile string `name:"message-file" short:"F" placeholder:"PATH" help:"read the tag message from PATH instead of generating it (- reads standard input)"`
	Expect      string `name:"expect" placeholder:"VERSION" help:"fail unless the bump yields VERSION"`
	DryRun      bool   `name:"dry-run" help:"check everything and print the tag without creating it"`
}

func (c *bumpCmd) Scope() *gitrepo.Scope { return c.ScopeArg }

// readMessage reads the tag message from path, or from standard input
// when path is "-".  An empty path means a generated message.
func readMessage(path string) (string, error) {
	var content []byte
	var err error
	switch path {
	case "":
		return "", nil
	case "-":
		content, err = io.ReadAll(os.Stdin)
	default:
		content, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read message file: %w", err)
	}

	message := string(content)
	if strings.TrimSpace(message) == "" {
		return "", errors.New("read message file: message is empty")
	}
	return message, nil
}

// parseExpect parses the version given to --expect.  It accepts the
// forms that tags use, so "1.2.3", "v1.2.3" and "mod/v1.2.3" all work.
func parseExpect(raw string) (*gitrepo.VersionSpec, error) {
	if raw == "" {
		return nil, nil
	}
	spec, err := gitrepo.ParseVersionSpec(raw)
	if err != nil {
		return nil, fmt.Errorf("parse --expect: %w", err)
	}
	return &spec, nil
}

func (c *bumpCmd) tagOptions() (bumper.TagOptions, error) {
	message, err := readMessage(c.MessageFile)
	if err != nil {
		return bumper.TagOptions{}, err
	}

	expect, err := parseExpect(c.Expect)
	if err != nil {
		return bumper.TagOptions{}, err
	}

	opts := bumper.TagOptions{Message: message}
	if expect != nil {
		opts.Expect = &expect.Version
	}
	return opts, nil
}

func (c *bumpCmd) Run(root *cli, repo *gitrepo.Context, head *plumbing.Reference) error {
	scope, err := effectiveScope(root, repo, c)
	if err != nil {
		return err
	}

	opts, err := c.tagOptions()
	if err != nil {
		return err
	}

	plan, err := bumper.PlanTag(repo, head, bumpParts[c.Part], scope, opts)
	if err != nil {
		return err
	}

	if c.DryRun {
		uiprint.Step("Would create annotated tag [%s]", plan.Label)
		uiprint.Substep("Target: %s", plan.Target)
		return nil
	}

	_, err = bumper.ApplyTag(repo, head, plan)
	return err
}
