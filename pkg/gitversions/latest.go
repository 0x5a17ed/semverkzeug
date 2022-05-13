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

package gitversions

import (
	"regexp"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

const (
	// FullVerPattern matches only against full versions, no pre-releases.
	FullVerPattern string = `\d+\.\d+\.\d+$`
)

var (
	FullVerRegexp = regexp.MustCompile(FullVerPattern)
)

type VString struct {
	Guide   gitrepo.Guide
	Prefix  string
	Version semver.Version
}

func (v VString) String() string {
	return v.Prefix + v.Version.String()
}

func Latest(repo *git.Repository, ref *plumbing.Reference) (*VString, error) {
	if ref == nil {
		vs := &VString{
			Prefix: "v", Version: *semver.MustParse("0.0.1-dev.0"),
		}
		return vs, nil
	}

	// Filter for versions without a pre-release component to make
	// sure that the distance to the last full version is measured
	// correctly and the next in-dev version shows a correct distance.
	guide, err := gitrepo.GetGuide(repo, ref, FullVerRegexp)
	if err != nil {
		return nil, err
	}

	vs := &VString{Guide: guide}
	if vts := FilterVersionTags(guide.Tags); len(vts) > 0 {
		vt := vts[len(vts)-1]
		vs.Prefix, vs.Version = vt.Prefix, vt.Version
	} else {
		// No version tags found.
		vs.Prefix, vs.Version = "v", *semver.MustParse("0.0.1-dev.0")
	}
	return vs, nil
}
