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
	"sort"

	"github.com/Masterminds/semver"
)

const (
	semVerPattern string = `(\d+)(\.\d+)?(\.\d+)?` +
		`(-([\dA-Za-z\-]+(\.[\dA-Za-z\-]+)*))?` +
		`(\+([\dA-Za-z\-]+(\.[\dA-Za-z\-]+)*))?$`
)

var (
	semVerRegExp = regexp.MustCompile(semVerPattern)
)

type VersionTag struct {
	Original string
	Prefix   string
	Version  semver.Version
}

func NewVersionTag(original string) (vt *VersionTag) {
	if loc := semVerRegExp.FindStringIndex(original); loc != nil {
		start, stop := loc[0], loc[1]
		if v, err := semver.NewVersion(original[start:stop]); err == nil {
			vt = &VersionTag{Original: original, Prefix: original[:start], Version: *v}
		}
	}
	return
}

type VersionTags []*VersionTag

func (s VersionTags) Len() int           { return len(s) }
func (s VersionTags) Less(i, j int) bool { return s[i].Version.LessThan(&s[j].Version) }
func (s VersionTags) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func FilterVersionTags(raw []string) (out VersionTags) {
	out = make(VersionTags, 0, len(raw))
	for _, r := range raw {
		if v := NewVersionTag(r); v != nil {
			out = append(out, v)
		}
	}
	sort.Sort(out)
	return
}
