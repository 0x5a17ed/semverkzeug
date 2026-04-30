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
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/internal/uiprint"
)

type Part int

const (
	_ Part = iota
	Major
	Minor
	Patch
)

var (
	ErrRepositoryIsEmpty = errors.New("repository is empty")
	ErrRepositoryIsDirty = errors.New("repository contains uncommitted changes")
)

func Bump(ov semver.Version, part Part) semver.Version {
	switch part {
	case Major:
		return ov.IncMajor()
	case Minor:
		return ov.IncMinor()
	case Patch:
		return ov.IncPatch()
	default:
		panic(fmt.Sprintf("bad part %d", part))
	}
}

func VerifyRepo(cx *gitrepo.Context, ref *plumbing.Reference) error {
	if ref == nil {
		return ErrRepositoryIsEmpty
	}

	st, err := gitrepo.BuildWorktreeStatus(cx)
	if err != nil {
		return fmt.Errorf("read worktree status: %w", err)
	}

	if !st.IsClean() {
		return ErrRepositoryIsDirty
	}

	return nil
}

func CreateTag(
	cx *gitrepo.Context,
	ref *plumbing.Reference,
	part Part,
	scope gitrepo.Scope,
) (*plumbing.Reference, error) {
	if err := VerifyRepo(cx, ref); err != nil {
		return nil, err
	}

	commit, err := cx.Repository().CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("resolve commit object: %w", err)
	}

	guide, err := gitrepo.BuildGuide(cx, ref, scope)
	if err != nil {
		return nil, fmt.Errorf("build guide: %w", err)
	}

	currSpec := gitrepo.LatestSpec(guide)
	nextLabel := currSpec.WithVersion(Bump(currSpec.Version, part)).String()

	// Double-check the tag label is not already in use.
	switch _, err := cx.Repository().Tag(nextLabel); {
	case errors.Is(err, git.ErrTagNotFound):
		// Fall through.
	case err != nil:
		return nil, fmt.Errorf("resolve tag %q: %w", nextLabel, err)
	default:
		return nil, fmt.Errorf("tag %q already exists", nextLabel)
	}

	uiprint.Step("Creating annotated tag %s", nextLabel)

	var message string
	if len(guide.Tags) == 0 {
		// No tags were used to find the previous version;
		// this means this is the first version to be tagged.
		message = fmt.Sprintf("first version %s", nextLabel)

	} else {
		message = fmt.Sprintf("bump version %s -> %s", currSpec.String(), nextLabel)
	}

	target, err := gitrepo.FindUniqueCommitHashAbbreviation(cx, commit)
	if err != nil {
		return nil, fmt.Errorf("abbreviate commit hash: %w", err)
	}
	subject, _, _ := strings.Cut(commit.Message, "\n")
	if subject = strings.TrimSpace(subject); subject != "" {
		target = fmt.Sprintf("%s (%s)", target, subject)
	}
	uiprint.Substep("Target: %s", target)

	var tagRef *plumbing.Reference

	// Check if the repository is backed by a filesystem storage.
	st, ok := cx.Repository().Storer.(*filesystem.Storage)
	if ok && st != nil && st.Filesystem() != nil {
		// Use the native git implementation to ensure consistency with other git commands.
		tagRef, err = createTagNative(cx, ref, nextLabel, message, st.Filesystem().Root())
		if err != nil {
			return nil, err
		}

	} else {
		// Fall back to tag creation via internal implementation.
		tagRef, err = createTagVirtual(cx, ref, nextLabel, message)
		if err != nil {
			return nil, err
		}
	}

	uiprint.Step("Created tag [%s]", tagRef.Name().Short())
	return tagRef, nil
}

// createTagVirtual creates a tag using the internal implementation.
func createTagVirtual(
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
	gitArgs := []string{"tag", "-a", "-F", "-", label, ref.Hash().String()}

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
