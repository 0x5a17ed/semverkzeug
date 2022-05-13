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

package bump

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
	"github.com/0x5a17ed/semverkzeug/pkg/gitversions"
)

type Part int

const (
	_ Part = iota
	Major
	Minor
	Patch
)

var (
	ErrRepositoryIsEmpty = errors.New("repository is empty")
	ErrRepositoryIsDirty = errors.New("repository contains uncommitted changes")
)

func Bump(ov *semver.Version, part Part) semver.Version {
	if ov == nil {
		ov = semver.MustParse("0.1.0")
	}

	switch part {
	case Major:
		return ov.IncMajor()
	case Minor:
		return ov.IncMinor()
	case Patch:
		return ov.IncPatch()
	default:
		panic(fmt.Sprintf("bad part %d", part))
	}
}

func VerifyRepo(repo *git.Repository, ref *plumbing.Reference) error {
	if ref == nil {
		return ErrRepositoryIsEmpty
	}

	if _, st, err := gitrepo.GetStatus(repo); err == nil {
		if !st.IsClean() {
			return ErrRepositoryIsDirty
		}
	} else {
		return err
	}
	return nil
}

func It(repo *git.Repository, ref *plumbing.Reference, part Part) (*plumbing.Reference, error) {
	if err := VerifyRepo(repo, ref); err != nil {
		return nil, err
	}

	oldVs, err := gitversions.Latest(repo, ref)
	if err != nil {
		return nil, err
	}

	newVs := gitversions.VString{
		Prefix: oldVs.Prefix, Version: Bump(&oldVs.Version, part),
	}

	var message string
	if len(oldVs.Guide.Tags) == 0 {
		// No tags were used to find the previous version,
		// this means this is the first version to be tagged.
		message = fmt.Sprintf("first version %s", newVs.String())
	} else {
		message = fmt.Sprintf("bump version %s → %s", oldVs.String(), newVs.String())
	}

	return repo.CreateTag(newVs.String(), ref.Hash(), &git.CreateTagOptions{Message: message})
}
