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

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/gitversions"
)

func Get(repo *git.Repository, ref *plumbing.Reference, addCommitHash bool) (vs *gitversions.VString, err error) {
	// Filter for versions without a pre-release component to make
	// sure that the distance to the last full version is measured
	// correctly and the next in-dev version shows a correct distance.
	if vs, err = gitversions.Latest(repo, ref); err != nil {
		return
	}

	wt, status, err := gitrepo.GetStatus(repo)
	if err != nil {
		return
	}

	mtime, err := gitversions.LastModificationTime(wt, status)
	if err != nil {
		if !errors.Is(err, gitversions.ErrNotDirty) {
			return
		}
		err = nil
	}

	if mtime != nil || vs.Guide.Depth != 0 {
		if vs.Version.Prerelease() == "" {
			vs.Version = vs.Version.IncPatch()
		}

		prerelease := fmt.Sprintf("dev.%d", vs.Guide.Depth)
		if mtime != nil {
			prerelease += "." + mtime.Format("20060102150405")
		}
		vs.Version, err = vs.Version.SetPrerelease(prerelease)
		if err != nil {
			return nil, err
		}

		if addCommitHash && !vs.Guide.Hash.IsZero() {
			vs.Version, err = vs.Version.SetMetadata("g" + vs.Guide.AbbreviatedHash())
			if err != nil {
				return nil, err
			}
		}
	}
	return vs, nil
}
