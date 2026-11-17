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

package bumper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/0x5a17ed/semverkzeug/internal/gitrepo"
	"github.com/0x5a17ed/semverkzeug/internal/uiprint"
)

func hasGitCommand() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// TagOptions adjust the tag that a bump plans.
type TagOptions struct {
	// Message replaces the generated tag message when it is not empty.
	Message string

	// Expect, when set, is the version that the bump must yield.  A
	// bump that yields any other version fails with
	// ErrUnexpectedVersion, so a caller can catch a disagreement with
	// its own record of versions before anything changes.
	Expect *semver.Version
}

// TagPlan describes the annotated tag that a bump creates.
type TagPlan struct {
	// Label is the tag name, such as "v1.2.3" or "mod/v1.2.3".
	Label string

	// Message is the tag message.
	Message string

	// Target describes the tagged commit for output: its unique
	// abbreviated hash, followed by its subject when it has one.
	Target string
}

// defaultMessage words the tag message that a bump creates when the
// caller gives none.
func defaultMessage(guide *gitrepo.Guide, current gitrepo.VersionSpec, label string) string {
	if len(guide.Tags) == 0 {
		// No tags were used to find the previous version;
		// this means this is the first version to be tagged.
		return fmt.Sprintf("first version %s", label)
	}
	return fmt.Sprintf("bump version %s -> %s", current.String(), label)
}

// describeTarget words the commit a tag points at as its unique
// abbreviated hash, followed by its subject when it has one.
func describeTarget(cx *gitrepo.Context, commit *object.Commit) (string, error) {
	target, err := gitrepo.FindUniqueCommitHashAbbreviation(cx, commit)
	if err != nil {
		return "", fmt.Errorf("abbreviate commit hash: %w", err)
	}

	subject, _, _ := strings.Cut(commit.Message, "\n")
	if subject = strings.TrimSpace(subject); subject != "" {
		target = fmt.Sprintf("%s (%s)", target, subject)
	}
	return target, nil
}

// PlanTag verifies the repository and works out the annotated tag that
// bumping part of the latest version creates at ref.  It creates
// nothing, so it also serves a dry run.
func PlanTag(
	cx *gitrepo.Context,
	ref *plumbing.Reference,
	part Part,
	scope gitrepo.Scope,
	opts TagOptions,
) (TagPlan, error) {
	if err := VerifyRepo(cx, ref); err != nil {
		return TagPlan{}, err
	}

	commit, err := cx.Repository().CommitObject(ref.Hash())
	if err != nil {
		return TagPlan{}, fmt.Errorf("resolve commit object: %w", err)
	}

	guide, err := gitrepo.BuildGuide(cx, ref, scope)
	if err != nil {
		return TagPlan{}, fmt.Errorf("build guide: %w", err)
	}

	currSpec := gitrepo.LatestSpec(guide)
	nextVersion := Bump(currSpec.Version, part)
	nextLabel := currSpec.WithVersion(nextVersion).String()
	if opts.Expect != nil && !nextVersion.Equal(opts.Expect) {
		return TagPlan{}, fmt.Errorf(
			"%w: bumping %s yields %s, not %s",
			ErrUnexpectedVersion, currSpec.String(), nextLabel, opts.Expect.String(),
		)
	}

	// Double-check the tag label is not already in use.
	switch _, err := cx.Repository().Tag(nextLabel); {
	case errors.Is(err, git.ErrTagNotFound):
		// Fall through.
	case err != nil:
		return TagPlan{}, fmt.Errorf("resolve tag %q: %w", nextLabel, err)
	default:
		return TagPlan{}, fmt.Errorf("tag %q already exists", nextLabel)
	}

	message := opts.Message
	if message == "" {
		message = defaultMessage(guide, currSpec, nextLabel)
	}

	target, err := describeTarget(cx, commit)
	if err != nil {
		return TagPlan{}, err
	}

	return TagPlan{
		Label:   nextLabel,
		Message: message,
		Target:  target,
	}, nil
}

// ApplyTag creates the annotated tag that plan describes at ref.
func ApplyTag(
	cx *gitrepo.Context,
	ref *plumbing.Reference,
	plan TagPlan,
) (*plumbing.Reference, error) {
	uiprint.Step("Creating annotated tag [%s]", plan.Label)
	uiprint.Substep("Target: %s", plan.Target)

	var tagRef *plumbing.Reference
	var err error

	// Check if the repository is backed by a filesystem storage.
	if p, hasStorage := cx.DotGitPath(); hasStorage && hasGitCommand() {
		// Use the native git implementation to ensure consistency with other git commands.
		tagRef, err = createTagNative(cx, ref, plan.Label, plan.Message, p)
	} else {
		// Fall back to tag creation via internal implementation.
		tagRef, err = createTagInternal(cx, ref, plan.Label, plan.Message)
	}
	if err != nil {
		return nil, err
	}

	uiprint.Step("Created tag [%s]", tagRef.Name().Short())
	return tagRef, nil
}

// CreateTag bumps part of the latest version and creates the annotated
// tag for it at ref.
func CreateTag(
	cx *gitrepo.Context,
	ref *plumbing.Reference,
	part Part,
	scope gitrepo.Scope,
	opts TagOptions,
) (*plumbing.Reference, error) {
	plan, err := PlanTag(cx, ref, part, scope, opts)
	if err != nil {
		return nil, err
	}
	return ApplyTag(cx, ref, plan)
}

// createTagInternal creates a tag using the internal implementation.
func createTagInternal(
	cx *gitrepo.Context,
	ref *plumbing.Reference,
	label string,
	message string,
) (*plumbing.Reference, error) {
	// Fall back to tag creation via internal implementation.
	tagRef, err := cx.Repository().CreateTag(label, ref.Hash(), &git.CreateTagOptions{
		Message: message,
	})
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	return tagRef, nil
}

// createTagNative creates a tag using the native git implementation.
func createTagNative(
	cx *gitrepo.Context,
	ref *plumbing.Reference,
	label string,
	message string,
	dotGit string,
) (*plumbing.Reference, error) {
	wtFs, err := cx.LoadWorktreeFilesystem()
	if err != nil {
		return nil, fmt.Errorf("get worktree filesystem: %w", err)
	}

	// Use the native git implementation to ensure consistency with other git commands.
	// Whitespace cleanup keeps lines that start with "#", such as
	// Markdown headings in release notes, which git otherwise strips
	// as comments.
	gitArgs := []string{"tag", "-a", "--cleanup=whitespace", "-F", "-", label, ref.Hash().String()}

	uiprint.Substep("Running: git %s", strings.Join(gitArgs, " "))

	cmd := exec.Command("git", gitArgs...)
	cmd.Env = append(
		slices.Clone(os.Environ()),
		fmt.Sprintf("GIT_DIR=%s", dotGit),
		fmt.Sprintf("GIT_WORK_TREE=%s", wtFs.Root()),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Allocate the stdin pipe that will be used to write the tag message.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("get stdin pipe: %w", err)
	}
	defer func() { _ = stdin.Close() }()

	// Start the command.
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	// If git takes more than a moment, surface a hint that it's
	// likely waiting on a hardware key or GPG passphrase.  Most of
	// the time the timer is cancelled before it fires.
	hintTimer := time.AfterFunc(2*time.Second, func() {
		uiprint.Hint("git may pause here waiting for signing (touch your security key if prompted)")
	})
	defer hintTimer.Stop()

	// Write the tag message to the stdin pipe.
	if err := feedMessage(stdin, message); err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	// Wait for the command to finish.
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("wait for command: %w", err)
	}

	tagRef, err := cx.Repository().Tag(label)
	if err != nil {
		return nil, fmt.Errorf("resolve tag %q: %w", label, err)
	}

	return tagRef, nil
}

func feedMessage(wr io.WriteCloser, message string) (err error) {
	defer func() {
		if errClose := wr.Close(); errClose != nil && err == nil {
			err = errClose
		}
	}()

	_, err = io.WriteString(wr, message)
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if sw, ok := wr.(interface{ Flush() error }); ok {
		return sw.Flush()
	}

	return nil
}
