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

var initialVersion = func() VersionSpec {
	vs, err := ParseVersionSpec("v0.0.1-dev.0")
	if err != nil {
		panic(err)
	}
	return vs
}()

func LatestSpec(g *Guide) VersionSpec {
	vt := g.HighestVersion()
	if vt == nil {
		// Return the initial version if there are no tags with
		// the given scope applied.
		return initialVersion.WithScope(g.Scope)
	}

	return vt.VersionSpec
}
