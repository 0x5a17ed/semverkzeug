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

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

func formatMTime(t *time.Time) string {
	ut := t.UTC()

	return fmt.Sprintf(
		"%s%02dZ",
		ut.Format("060102T150405"),
		ut.Nanosecond()/10_000_000,
	)
}

// Describe returns a floating version string for the given reference.
func Describe(
	cx *gitrepo.Context,
	guide *gitrepo.Guide,
) (gitrepo.VersionSpec, error) {
	mtime, err := gitrepo.FindWorktreeMTime(cx)
	switch {
	case errors.Is(err, git.ErrIsBareRepository):
		err = nil // Ignore.
	case errors.Is(err, gitrepo.ErrWorktreeClean):
		err = nil // Ignore.
	case err != nil:
		return gitrepo.VersionSpec{}, fmt.Errorf("find worktree mtime: %w", err)
	default:
		// Fall through.
	}

	spec := gitrepo.LatestSpec(guide)

	// Return the latest version if there are no changes.
	if mtime == nil && guide.IsPure() {
		return spec, nil
	}

	// Bump the patch version if there are no pre-releases.
	if spec.Version.Prerelease() == "" {
		spec.Version = spec.Version.IncPatch()
	}

	// Use last commit time as the timestamp if there are no changes.
	if mtime == nil && guide.HasCommit() {
		mtime = &guide.Commit.Committer.When
	}

	// Set the prerelease version to "dev" and the timestamp.
	devCounter := "0"
	if mtime != nil {
		devCounter = formatMTime(mtime)
	}
	prerelease := fmt.Sprintf("dev.%s", devCounter)

	spec.Version, err = spec.Version.SetPrerelease(prerelease)
	if err != nil {
		return gitrepo.VersionSpec{}, err
	}

	return spec, nil
}
