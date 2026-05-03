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

package bumper

import (
	"github.com/Masterminds/semver/v3"
)

type partFunc func(semver.Version) semver.Version

func (f partFunc) bump(inp semver.Version) semver.Version {
	return f(inp)
}

type Part interface {
	bump(inp semver.Version) semver.Version
}

var (
	Major Part = partFunc(semver.Version.IncMajor)
	Minor Part = partFunc(semver.Version.IncMinor)
	Patch Part = partFunc(semver.Version.IncPatch)
)

// Bump calculates a new semantic version by incrementing the
// specified part of the provided version.
func Bump(ov semver.Version, part Part) semver.Version {
	return part.bump(ov)
}
