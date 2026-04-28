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

	"github.com/go-git/go-git/v5/plumbing"
)

var initialVersion = func() VersionSpec {
	vs, err := ParseVersionSpec("v0.0.1-dev.0")
	if err != nil {
		panic(err)
	}
	return vs
}()

type VersionState struct {
	Spec VersionSpec

	// Guide explains how to reach the current version.
	Guide *Guide
}

func (v VersionState) String() string {
	return fmt.Sprintf("%s", v.Spec)
}

// HasGuide reports whether v has a version guide.
func (v VersionState) HasGuide() bool {
	return v.Guide != nil
}

// IsPure reports whether v identifies a tagged commit exactly,
// rather than a developmental snapshot some distance past it.
func (v VersionState) IsPure() bool {
	return v.HasGuide() && len(v.Guide.Tags) > 0 && v.Guide.Depth == 0
}

// FindLatestVersion returns the latest version of the repo.
func FindLatestVersion(gCx *Context, ref *plumbing.Reference, scope Scope) (*VersionState, error) {
	guide, err := NewGuide(gCx, ref, scope)
	if err != nil {
		return nil, fmt.Errorf("build version guide: %w", err)
	}

	if len(guide.Tags) == 0 {
		// No version tags found.
		return &VersionState{
			Spec:  initialVersion.WithScope(scope),
			Guide: guide,
		}, nil
	}

	// Select the best version tag.
	vtc := SelectHighestVersionTag(guide.Tags)

	return &VersionState{
		Spec:  vtc.VersionSpec.WithScope(scope),
		Guide: guide,
	}, nil
}
