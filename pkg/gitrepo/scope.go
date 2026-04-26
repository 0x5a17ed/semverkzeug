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
	"path"
)

// Scope represents a cleaned, validated tag scope — a relative
// unix-style path that identifies which module or sub-directory a
// version tag belongs to.
//
// The zero value is the root scope (matching tags without a scope
// prefix, e.g. "v1.0.0").  A non-root scope like "mod" matches
// tags prefixed with that path (e.g. "mod/v1.0.0").
type Scope struct {
	path string
}

// RootScope returns the root scope (tags without a scope prefix).
func RootScope() Scope { return Scope{} }

// ParseScope validates and normalises a raw scope string.
//
// It trims whitespace and slashes, collapses path separators, and
// rejects absolute paths or paths that escape the repository root
// via "..".  The inputs "", ".", and "/" all resolve to the root
// scope.
func ParseScope(raw string) (Scope, error) {
	// Use path.Clean (unix-style) to collapse redundant separators
	// and resolve single-dot segments.
	s := path.Clean(raw)

	// "." and "" both mean root.
	if s == "" || s == "." {
		return Scope{}, nil
	}

	if !scopeRegExp.MatchString(s) {
		return Scope{}, fmt.Errorf("%#q: invalid scope format", raw)
	}

	return Scope{path: s}, nil
}

// String returns the normalised scope path.  The root scope is
// represented as the empty string.
func (s Scope) String() string {
	return s.path
}

// IsRoot reports whether s is the root scope.
func (s Scope) IsRoot() bool { return s.path == "" }

// Matches reports whether a tag's parsed scope field equals this
// scope.  VersionSpec.Scope is "" for unscoped tags and e.g. "mod"
// for "mod/v1.0.0".
func (s Scope) Matches(tagScope Scope) bool {
	return s.path == tagScope.path
}
