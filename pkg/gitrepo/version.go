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

package gitrepo

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var initialVersion = func() *VersionSpec {
	vs, err := ParseVersionSpec("v0.0.1-dev.0")
	if err != nil {
		panic(err)
	}
	return vs
}()

type LatestVersion struct {
	Spec VersionSpec

	// Guide explains how to reach the current version.
	Guide *Guide
}

func (v LatestVersion) String() string {
	return fmt.Sprintf("%s", v.Spec)
}

// FindLatestVersion returns the latest version of the repo.
func FindLatestVersion(repo *git.Repository, ref *plumbing.Reference, scope Scope) (*LatestVersion, error) {
	guide, err := NewGuide(repo, ref, scope)
	if err != nil {
		return nil, fmt.Errorf("build version guide: %w", err)
	}

	if len(guide.Tags) == 0 {
		// No version tags found.
		return &LatestVersion{
			Spec:  initialVersion.WithScope(scope),
			Guide: guide,
		}, nil
	}

	// Select the best version tag.
	vtc := SelectHighestVersionTag(guide.Tags)

	return &LatestVersion{
		Guide: guide,
		Spec:  vtc.VersionSpec.WithScope(scope),
	}, nil
}
