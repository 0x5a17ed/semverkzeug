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

package gitrepo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

const commentPrefix = "#"

type ignoreMatcher struct {
	matcher    gitignore.Matcher
	ignoreCase bool
}

func loadIgnoreMatcher(cx *Context) (*ignoreMatcher, error) {
	wtFsys, err := cx.LoadWorktreeFilesystem()
	if err != nil {
		return nil, fmt.Errorf("load worktree filesystem: %w", err)
	}

	cfg, err := loadEffectiveConfig(cx)
	if err != nil {
		return nil, err
	}

	var patterns []gitignore.Pattern

	var excludesFile string
	if cfg.excludesFile == nil {
		excludesFile, err = defaultExcludesFilePath()
		if err != nil {
			return nil, err
		}
	} else {
		excludesFile = *cfg.excludesFile
	}

	if excludesFile != "" {
		ps, err := readExcludesFilePatterns(wtFsys, excludesFile, cfg.ignoreCase)
		if err != nil {
			return nil, fmt.Errorf("load core.excludesFile patterns: %w", err)
		}
		patterns = append(patterns, ps...)
	}

	ps, err := readInfoExcludePatterns(cx, cfg.ignoreCase)
	if err != nil {
		return nil, err
	}
	patterns = append(patterns, ps...)

	ps, err = readWorktreeIgnorePatterns(wtFsys, "", cfg.ignoreCase)
	if err != nil {
		return nil, err
	}
	patterns = append(patterns, ps...)

	if len(patterns) == 0 {
		return &ignoreMatcher{ignoreCase: cfg.ignoreCase}, nil
	}

	return &ignoreMatcher{
		matcher:    gitignore.NewMatcher(patterns),
		ignoreCase: cfg.ignoreCase,
	}, nil
}

func (m *ignoreMatcher) Match(path string, isDir bool) bool {
	if m == nil || m.matcher == nil {
		return false
	}

	parts := splitGitPath(path, m.ignoreCase)
	if len(parts) == 0 {
		return false
	}

	for i := 1; i < len(parts); i++ {
		if m.matcher.Match(parts[:i], true) {
			return true
		}
	}

	return m.matcher.Match(parts, isDir)
}

func readExcludesFilePatterns(
	wtFsys billy.Filesystem,
	p string,
	ignoreCase bool,
) ([]gitignore.Pattern, error) {
	expanded, err := expandTilde(p)
	if err != nil {
		return nil, err
	}

	if filepath.IsAbs(expanded) {
		return readIgnoreFilePatterns(osfs.New("/"), expanded, nil, ignoreCase)
	}

	return readIgnoreFilePatterns(wtFsys, filepath.ToSlash(expanded), nil, ignoreCase)
}

func readInfoExcludePatterns(
	cx *Context,
	ignoreCase bool,
) ([]gitignore.Pattern, error) {
	fsys := cx.DotGitFilesystem()
	if fsys == nil {
		return nil, nil
	}

	ps, err := readIgnoreFilePatterns(fsys, "info/exclude", nil, ignoreCase)
	if err != nil {
		return nil, fmt.Errorf("load .git/info/exclude: %w", err)
	}
	return ps, nil
}

func readWorktreeIgnorePatterns(
	wtFsys billy.Filesystem,
	dir string,
	ignoreCase bool,
) ([]gitignore.Pattern, error) {
	var patterns []gitignore.Pattern

	ignorePath := wtFsys.Join(dir, ".gitignore")
	if isRegularFile(wtFsys, ignorePath) {
		ps, err := readIgnoreFilePatterns(wtFsys, ignorePath, splitGitPath(dir, ignoreCase), ignoreCase)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, ps...)
	}

	entries, err := wtFsys.ReadDir(dir)
	switch {
	case isErrNotExist(err):
		return patterns, nil
	case err != nil:
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".git" {
			continue
		}

		p := wtFsys.Join(dir, entry.Name())
		if isSymlink(wtFsys, p) {
			continue
		}

		ps, err := readWorktreeIgnorePatterns(wtFsys, p, ignoreCase)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, ps...)
	}

	return patterns, nil
}

func readIgnoreFilePatterns(
	fsys billy.Filesystem,
	path string,
	domain []string,
	ignoreCase bool,
) ([]gitignore.Pattern, error) {
	f, err := fsys.Open(path)
	switch {
	case isErrNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}
	defer func() { _ = f.Close() }()

	if ignoreCase {
		domain = lowerStrings(domain)
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var patterns []gitignore.Pattern
	for sc.Scan() {
		line := sc.Text()
		switch {
		case len(line) == 0:
			continue
		case strings.HasPrefix(line, commentPrefix):
			continue
		}
		if ignoreCase {
			line = strings.ToLower(line)
		}
		patterns = append(patterns, gitignore.ParsePattern(line, domain))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

func splitGitPath(p string, ignoreCase bool) []string {
	p = filepath.ToSlash(filepath.Clean(p))
	if p == "." || p == "/" {
		return nil
	}
	p = strings.TrimPrefix(p, "/")
	if ignoreCase {
		p = strings.ToLower(p)
	}
	return strings.Split(p, "/")
}

func lowerStrings(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToLower(s)
	}
	return out
}

func isRegularFile(fsys billy.Filesystem, p string) bool {
	info, err := fsys.Lstat(p)
	return err == nil && info.Mode()&os.ModeSymlink == 0 && !info.IsDir()
}

func isSymlink(fsys billy.Filesystem, p string) bool {
	info, err := fsys.Lstat(p)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
