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
	"regexp"

	"github.com/Masterminds/semver/v3"
)

const (
	scopePattern = `[A-Za-z0-9][A-Za-z0-9._-]*(?:/[A-Za-z0-9][A-Za-z0-9._-]*)*`

	// Optional scope before the version, like "foo/".
	scopePart = `(?:(?P<scope>` + scopePattern + `)/)?`

	// Optional "v" prefix, exposed separately.
	prefixPart = `(?P<prefix>v|[A-Za-z][A-Za-z0-9._-]*)?`

	// SemVer numeric identifier: "0" or non-zero digit followed by digits.
	numericIdentifier = `(0|[1-9]\d*)`

	majorPart = `(?P<major>` + numericIdentifier + `)`
	minorPart = `(?P<minor>` + numericIdentifier + `)`
	patchPart = `(?P<patch>` + numericIdentifier + `)`

	// Pre-release identifier:
	// - Numeric: 0 or non-zero-leading integer
	// - Or alphanumeric/hyphen containing at least one letter/hyphen
	preReleaseIdentifier = `(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)`

	// Dot-separated pre-release identifiers.
	preReleasePart = `(?:-(?P<prerelease>` +
		preReleaseIdentifier +
		`(?:\.` + preReleaseIdentifier + `)*))?`

	// Dot-separated build identifiers.
	buildPart = `(?:\+(?P<buildmetadata>[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?`

	// coreVersionPart is the semver itself, excluding optional scope and excluding optional "v".
	coreVersionPart = `(?P<coreversion>` +
		majorPart + `\.` + minorPart + `\.` + patchPart +
		preReleasePart +
		buildPart +
		`)`

	// versionPart is what appeared in the input after the optional scope,
	// including the optional "v" prefix.
	versionPart = `(?P<version>` +
		prefixPart +
		coreVersionPart +
		`)`

	scopedSemVerPattern = scopePart + versionPart
)

var (
	scopeRegExp  = regexp.MustCompile(`^` + scopePattern + `$`)
	semVerRegExp = regexp.MustCompile(`^` + scopedSemVerPattern + `$`)
)

func parse(s string) map[string]string {
	m := semVerRegExp.FindStringSubmatch(s)
	if m == nil {
		return nil
	}

	out := make(map[string]string, len(m))
	for i, name := range semVerRegExp.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		out[name] = m[i]
	}
	return out
}

// VersionSpec represents a version tag string.
type VersionSpec struct {
	// Scope is the optional scope before the version, like `foo/`.
	Scope Scope

	// Prefix is the optional "v" prefix, exposed separately.
	Prefix string

	// Version is the semver itself, excluding optional scope and excluding optional "v".
	Version semver.Version
}

func (s VersionSpec) String() string {
	if s.Scope.IsRoot() {
		return s.Prefix + s.Version.String()
	}
	return s.Scope.String() + "/" + s.Prefix + s.Version.String()
}

func (s VersionSpec) WithScope(scope Scope) VersionSpec {
	return VersionSpec{
		Scope:   scope,
		Prefix:  s.Prefix,
		Version: s.Version,
	}
}

// ParseVersionSpec parses a version tag string into a VersionSpec.
func ParseVersionSpec(original string) (*VersionSpec, error) {
	m := parse(original)
	if m == nil {
		return nil, fmt.Errorf("%#q: invalid version tag format", original)
	}

	s, err := ParseScope(m["scope"])
	if err != nil {
		return nil, fmt.Errorf("%#q: invalid scope format", original)
	}

	v, err := semver.NewVersion(m["version"])
	if err != nil {
		return nil, fmt.Errorf("%#q: invalid version tag format", original)
	}

	vt := &VersionSpec{
		Scope:   s,
		Prefix:  m["prefix"],
		Version: *v,
	}

	return vt, nil
}
