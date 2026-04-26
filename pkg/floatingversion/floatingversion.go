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

package floatingversion

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

func formatMTime(t *time.Time) string {
	ut := t.UTC()

	return fmt.Sprintf(
		"%s%02dZ",
		t.UTC().Format("060102T150405"),
		ut.Nanosecond()/10_000_000,
	)
}

// Describe returns a floating version string for the given reference.
func Describe(
	repo *git.Repository,
	ref *plumbing.Reference,
	scope gitrepo.Scope,
) (vs *gitrepo.LatestVersion, err error) {
	wt, status, err := gitrepo.GetStatus(repo)
	if err != nil {
		return nil, fmt.Errorf("get worktree status: %w", err)
	}

	mtime, err := gitrepo.FindWorktreeMTime(wt, status)
	switch {
	case errors.Is(err, gitrepo.ErrNotDirty):
		err = nil // Ignore.
	case err != nil:
		return nil, fmt.Errorf("find worktree mtime: %w", err)
	}

	if vs, err = gitrepo.FindLatestVersion(repo, ref, scope); err != nil {
		return nil, fmt.Errorf("find latest version: %w", err)
	}

	// Return the latest version if there are no changes.
	if mtime == nil && vs.Guide.Depth == 0 {
		return vs, nil
	}

	// Bump the patch version if there are no pre-releases.
	if vs.Spec.Version.Prerelease() == "" {
		vs.Spec.Version = vs.Spec.Version.IncPatch()
	}

	// Use last commit time as the timestamp if there are no changes.
	if mtime == nil && vs.Guide.HasCommit() {
		mtime = &vs.Guide.Commit.Committer.When
	}

	// Set the prerelease version to "dev" and the timestamp.
	prerelease := fmt.Sprintf("dev")
	if mtime != nil {
		prerelease += "." + formatMTime(mtime)
	}
	vs.Spec.Version, err = vs.Spec.Version.SetPrerelease(prerelease)
	if err != nil {
		return nil, err
	}

	return vs, nil
}
