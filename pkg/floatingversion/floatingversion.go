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

package floatingversion

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/go-git/go-git/v5"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

var devLabelRegexp = regexp.MustCompile(`(^|\.)dev(\.[0-9][0-9A-Za-z]*)*(\.|$)`)

func upsertDev(pre, newDev string) string {
	m := devLabelRegexp.FindStringSubmatchIndex(pre)
	if m == nil {
		if pre == "" {
			return newDev
		}
		return pre + "." + newDev
	}

	// m[3] = end of left boundary; m[6] = start of right boundary.
	// Slicing through m[6] keeps the trailing "." (or empty) intact.
	return pre[:m[3]] + newDev + pre[m[6]:]
}

func formatMTime(t *time.Time) string {
	if t == nil {
		return "0"
	}

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
	mtime, err := gitrepo.FindStableWorktreeMTime(cx)
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

	if mtime == nil {
		// Return the latest version if there are no changes since the last tag.
		if guide.IsPure() {
			return spec, nil
		}

		// Try filling mtime from the timestamp of the last commit.
		if guide.HasCommit() {
			mtime = &guide.Commit.Committer.When
		}

		// mtime still can be nil here, if the repo is empty, for example.
	}

	prereleaseLabel := spec.Version.Prerelease()

	// Bump the patch version if the spec is not a pre-release yet.
	if prereleaseLabel == "" {
		spec.Version = spec.Version.IncPatch()
	}

	// Set the prerelease version to "dev" and the timestamp.
	newDevLabel := fmt.Sprintf("dev.%s", formatMTime(mtime))

	prereleaseLabel = upsertDev(prereleaseLabel, newDevLabel)

	spec.Version, err = spec.Version.SetPrerelease(prereleaseLabel)
	if err != nil {
		return gitrepo.VersionSpec{}, fmt.Errorf("set prerelease: %w", err)
	}

	return spec, nil
}
