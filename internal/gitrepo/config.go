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
	"bytes"
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/gcfg"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	billyutil "github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	formatconfig "github.com/go-git/go-git/v5/plumbing/format/config"
)

const maxConfigIncludeDepth = 10

type effectiveConfig struct {
	excludesFile   *string
	ignoreCase     bool
	worktreeConfig bool
}

type configLoader struct {
	repo         *git.Repository
	cfg          effectiveConfig
	rootFS       billy.Filesystem
	dotGitFS     billy.Filesystem
	worktreeRoot string
	seen         map[string]int
}

type configFile struct {
	fsys billy.Filesystem
	path string
	key  string
}

func loadEffectiveConfig(cx *Context) (effectiveConfig, error) {
	wtr, err := cx.LoadWorktreeRoot()
	if err != nil {
		return effectiveConfig{}, fmt.Errorf("load worktree root: %w", err)
	}

	loader := &configLoader{
		repo:         cx.Repository(),
		rootFS:       osfs.New("/"),
		dotGitFS:     cx.DotGitFilesystem(),
		worktreeRoot: wtr,
		seen:         map[string]int{},
	}

	for _, p := range systemConfigPaths() {
		if err := loader.applyConfigFile(configFile{
			fsys: loader.rootFS,
			path: p,
			key:  "system:" + p,
		}, 0); err != nil {
			return effectiveConfig{}, err
		}
	}

	for _, p := range globalConfigPaths() {
		if err := loader.applyConfigFile(configFile{
			fsys: loader.rootFS,
			path: p,
			key:  "global:" + p,
		}, 0); err != nil {
			return effectiveConfig{}, err
		}
	}

	if loader.dotGitFS != nil {
		if err := loader.applyConfigFile(configFile{
			fsys: loader.dotGitFS,
			path: "config",
			key:  "local:config",
		}, 0); err != nil {
			return effectiveConfig{}, err
		}

		if loader.cfg.worktreeConfig {
			if err := loader.applyConfigFile(configFile{
				fsys: loader.dotGitFS,
				path: "config.worktree",
				key:  "worktree:config.worktree",
			}, 0); err != nil {
				return effectiveConfig{}, err
			}
		}
	} else if cx.Repository() != nil {
		cfg, err := cx.Repository().Config()
		if err != nil {
			return effectiveConfig{}, fmt.Errorf("load repository config: %w", err)
		}
		loader.applyRawConfig(cfg.Raw)
	}

	if err := loader.applyCommandConfigEnv(); err != nil {
		return effectiveConfig{}, err
	}

	return loader.cfg, nil
}

func (l *configLoader) configWalker(
	cf configFile,
	depth int,
	section, subsection, key, value string,
	boolValue bool,
) error {
	if subsection == "" && key == "" {
		return nil
	}

	if strings.EqualFold(section, "include") && subsection == "" && strings.EqualFold(key, "path") {
		next, err := l.resolveInclude(cf, value)
		if err != nil {
			return err
		}
		return l.applyConfigFile(next, depth+1)
	}

	if strings.EqualFold(section, "includeIf") && strings.EqualFold(key, "path") {
		ok, err := l.includeIfMatches(subsection)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		next, err := l.resolveInclude(cf, value)
		if err != nil {
			return err
		}
		return l.applyConfigFile(next, depth+1)
	}

	return l.applyConfigOption(section, subsection, key, value, boolValue)
}

func (l *configLoader) applyConfigFile(cf configFile, depth int) error {
	if cf.fsys == nil || cf.path == "" {
		return nil
	}
	if depth > maxConfigIncludeDepth {
		return fmt.Errorf("git config include depth exceeds %d at %s", maxConfigIncludeDepth, cf.path)
	}

	key := cf.key
	if key == "" {
		key = cf.path
	}
	l.seen[key]++
	defer func() { l.seen[key]-- }()
	if l.seen[key] > maxConfigIncludeDepth {
		return fmt.Errorf("git config include cycle involving %s", cf.path)
	}

	b, err := billyutil.ReadFile(cf.fsys, cf.path)
	switch {
	case isErrNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("read git config %s: %w", cf.path, err)
	}

	walker := func(section, subsection, key, value string, boolValue bool) error {
		return l.configWalker(cf, depth, section, subsection, key, value, boolValue)
	}
	return gcfg.ReadWithCallback(bytes.NewReader(b), walker)
}

func (l *configLoader) applyRawConfig(raw *formatconfig.Config) {
	if raw == nil {
		return
	}

	core := raw.Section("core")
	if core.HasOption("excludesfile") {
		l.cfg.excludesFile = new(core.Option("excludesfile"))
	}
	if core.HasOption("ignorecase") {
		if b, err := parseGitBool(core.Option("ignorecase"), false); err == nil {
			l.cfg.ignoreCase = b
		}
	}

	extensions := raw.Section("extensions")
	if extensions.HasOption("worktreeconfig") {
		if b, err := parseGitBool(extensions.Option("worktreeconfig"), false); err == nil {
			l.cfg.worktreeConfig = b
		}
	}
}

func (l *configLoader) applyConfigOption(section, subsection, key, value string, boolValue bool) error {
	if subsection != "" {
		return nil
	}

	switch {
	case strings.EqualFold(section, "core") && strings.EqualFold(key, "excludesfile"):
		l.cfg.excludesFile = new(value)
	case strings.EqualFold(section, "core") && strings.EqualFold(key, "ignorecase"):
		b, err := parseGitBool(value, boolValue)
		if err != nil {
			return fmt.Errorf("parse core.ignoreCase: %w", err)
		}
		l.cfg.ignoreCase = b
	case strings.EqualFold(section, "extensions") && strings.EqualFold(key, "worktreeconfig"):
		b, err := parseGitBool(value, boolValue)
		if err != nil {
			return fmt.Errorf("parse extensions.worktreeConfig: %w", err)
		}
		l.cfg.worktreeConfig = b
	}

	return nil
}

func (l *configLoader) applyCommandConfigEnv() error {
	countText := os.Getenv("GIT_CONFIG_COUNT")
	if countText == "" {
		return nil
	}

	count, err := strconv.Atoi(countText)
	if err != nil {
		return fmt.Errorf("parse GIT_CONFIG_COUNT: %w", err)
	}

	for i := 0; i < count; i++ {
		name := os.Getenv(fmt.Sprintf("GIT_CONFIG_KEY_%d", i))
		value := os.Getenv(fmt.Sprintf("GIT_CONFIG_VALUE_%d", i))

		section, key, ok := strings.Cut(name, ".")
		if !ok {
			continue
		}
		if err := l.applyConfigOption(section, "", key, value, false); err != nil {
			return err
		}
	}

	return nil
}

func (l *configLoader) resolveInclude(parent configFile, includePath string) (configFile, error) {
	p, err := expandTilde(includePath)
	if err != nil {
		return configFile{}, fmt.Errorf("expand include.path %q: %w", includePath, err)
	}

	if filepath.IsAbs(p) {
		return configFile{
			fsys: l.rootFS,
			path: p,
			key:  "include:" + p,
		}, nil
	}

	base := filepath.Dir(parent.path)
	next := parent.fsys.Join(base, p)
	return configFile{
		fsys: parent.fsys,
		path: next,
		key:  parent.key + "->" + next,
	}, nil
}

func (l *configLoader) includeIfMatches(condition string) (bool, error) {
	kind, pattern, ok := strings.Cut(condition, ":")
	if !ok {
		return false, nil
	}

	switch strings.ToLower(kind) {
	case "gitdir", "gitdir/i":
		target := dotGitRoot(l.dotGitFS)
		if target == "" {
			target = l.worktreeRoot
		}
		pattern, err := expandTildeInGitdirPattern(pattern)
		if err != nil {
			return false, err
		}
		return matchConfigGlob(pattern, target, strings.EqualFold(kind, "gitdir/i")), nil
	case "onbranch":
		if l.repo == nil {
			return false, nil
		}
		ref, err := l.repo.Head()
		if err != nil {
			return false, nil
		}
		return matchConfigGlob(pattern, ref.Name().Short(), false), nil
	default:
		return false, nil
	}
}

func systemConfigPaths() []string {
	if os.Getenv("GIT_CONFIG_NOSYSTEM") != "" {
		return nil
	}
	if p := os.Getenv("GIT_CONFIG_SYSTEM"); p != "" {
		return []string{p}
	}
	return []string{"/etc/gitconfig"}
}

func globalConfigPaths() []string {
	if p := os.Getenv("GIT_CONFIG_GLOBAL"); p != "" {
		return []string{p}
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}

	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}

	return []string{
		filepath.Join(xdg, "git", "config"),
		filepath.Join(home, ".gitconfig"),
	}
}

func defaultExcludesFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if home == "" {
		return "", fmt.Errorf("home directory is empty")
	}

	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}

	return filepath.Join(xdg, "git", "ignore"), nil
}

func parseGitBool(value string, boolValue bool) (bool, error) {
	if boolValue {
		return true, nil
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid git bool %q", value)
	}
}

func expandTildeInGitdirPattern(p string) (string, error) {
	hadTrailingSlash := strings.HasSuffix(p, "/")

	expanded, err := expandTilde(p)
	if err != nil {
		return "", err
	}

	if hadTrailingSlash && !strings.HasSuffix(filepath.ToSlash(expanded), "/") {
		expanded += "/"
	}

	return expanded, nil
}

func matchConfigGlob(pattern, target string, ignoreCase bool) bool {
	pattern = filepath.ToSlash(pattern)
	target = filepath.ToSlash(target)
	if ignoreCase {
		pattern = strings.ToLower(pattern)
		target = strings.ToLower(target)
	}

	if strings.HasSuffix(pattern, "/") {
		pattern += "**"
	}
	if !filepath.IsAbs(pattern) && !strings.HasPrefix(pattern, "**/") {
		pattern = "**/" + pattern
	}

	ok, err := doublestar.Match(pattern, target)
	return err == nil && ok
}

func dotGitRoot(fsys billy.Filesystem) string {
	return filesystemRoot(fsys)
}

func filesystemRoot(fsys billy.Filesystem) string {
	type rooter interface {
		Root() string
	}

	r, ok := fsys.(rooter)
	if !ok || r == nil {
		return ""
	}

	return r.Root()
}

func isErrNotExist(err error) bool {
	return errors.Is(err, iofs.ErrNotExist) || os.IsNotExist(err)
}
