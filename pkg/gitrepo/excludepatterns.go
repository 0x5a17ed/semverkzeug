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
	"bufio"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"go.uber.org/multierr"
)

const (
	commentPrefix = "#"
)

func readIgnoreFile(bfs billy.Filesystem, filePath string) (ps []gitignore.Pattern, err error) {
	f, err := bfs.Open(filePath)
	if err != nil {
		return
	}
	defer multierr.AppendInvoke(&err, multierr.Close(f))

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if !strings.HasPrefix(s, commentPrefix) && len(strings.TrimSpace(s)) > 0 {
			ps = append(ps, gitignore.ParsePattern(s, nil))
		}
	}
	return
}

func loadGlobalExcludePatterns() (out []gitignore.Pattern, err error) {
	config, err := gitconfig.LoadConfig(gitconfig.GlobalScope)
	if err != nil {
		return out, err
	}

	excludesFiles := config.Raw.Section("core").OptionAll("excludesfile")
	for _, excludesFile := range excludesFiles {
		ps, err := readIgnoreFile(osfs.New("/"), excludesFile)
		if err != nil {
			return nil, err
		}
		out = append(out, ps...)
	}
	return
}
