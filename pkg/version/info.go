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

package version

import (
	"runtime/debug"
	"slices"
)

var Version = "dev"

func initVersion() {
	if Version != "" {
		return
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if !slices.Contains([]string{"", "(devel)"}, bi.Main.Version) {
		Version = bi.Main.Version
	}
}

func init() {
	initVersion()
}
