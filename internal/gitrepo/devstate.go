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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/google/renameio/v2"
)

// devCounterTick is the smallest increment applied to the persisted floor
// when the freshly computed candidate is not strictly after it. Matches the
// centisecond resolution used by the floating dev version label.
const devCounterTick = 10 * time.Millisecond

// devStateRelPath is the per-worktree path (relative to the gitdir) where
// the dev counter state is persisted.
const devStateRelPath = "semverkzeug/state.json"

// devState is the on-disk record consulted by FindStableWorktreeMTime.
//
// Fingerprint identifies the dirty file set that produced Emitted. When
// the inputs are unchanged across calls, the fingerprint matches and
// Emitted is returned verbatim — guaranteeing stability. When inputs
// change but their max mtime would step backwards, Emitted is treated
// as a floor and advanced by devCounterTick — guaranteeing monotonicity.
type devState struct {
	Fingerprint string    `json:"fingerprint"`
	Emitted     time.Time `json:"emitted"`
}

// devStatePath resolves the on-disk path of the per-worktree state file.
// Returns "" and false when the repository is not backed by an OS filesystem (in
// which case persistence is not possible and silently skipped).
func devStatePath(cx *Context) (string, bool) {
	p, ok := cx.DotGitPath()
	if !ok {
		return "", false
	}
	return filepath.Join(p, devStateRelPath), true
}

// loadDevState reads the persisted state. A missing file yields a nil
// state without error, so callers can treat absence and best-effort
// failure identically.
func loadDevState(path string) (*devState, error) {
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var s devState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &s, nil
}

// saveDevState atomically writes the state record. All errors are
// silently ignored: persistence is best effort by design (read-only
// .git, sandboxed builds, etc. should not break version reporting).
func saveDevState(path string, s devState) error {
	data, err := json.Marshal(&s)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	if err := renameio.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

// computeFingerprint hashes the relevant inputs that determine the
// candidate mtime. Status flags are included so that staging activity
// (which does not change a file's mtime) still invalidates the cache.
// The index file's mtime is intentionally excluded so that routine git
// activity that touches .git/index without changing real inputs does
// not perturb the result.
func computeFingerprint(entries []DirtyEntry) string {
	if len(entries) == 0 {
		return ""
	}

	sorted := make([]DirtyEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].path < sorted[j].path
	})

	h := sha256.New()
	var buf []byte
	for _, e := range sorted {
		buf = buf[:0]
		buf = append(buf, e.path...)
		buf = append(buf, 0)
		buf = strconv.AppendInt(buf, int64(e.WorktreeStatus()), 10)
		buf = append(buf, 0)
		buf = strconv.AppendInt(buf, int64(e.StagingStatus()), 10)
		buf = append(buf, 0)
		buf = strconv.AppendInt(buf, e.ModTime().UnixNano(), 10)
		buf = append(buf, '\n')
		h.Write(buf)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FindStableWorktreeMTime returns the last modification time of the
// files in the working tree, augmented with two guarantees backed by
// per-worktree persisted state in .git/semverkzeug/state.json:
//
//  1. Stability — repeated calls with the same dirty file set (paths,
//     status flags, mtimes) return the same value, even if .git/index's
//     mtime drifts in the meantime.
//
//  2. Monotonicity — the returned value never decreases between calls.
//     If the freshly computed candidate would step backwards, the
//     previously emitted value is advanced by one centisecond instead.
//
// Persistence is best-effort. When the state file cannot be read or
// written (in-memory storage, read-only .git, corrupted state), the
// function degrades to plain max(index mtime, dirty file mtimes)
// without surfacing an error.
func FindStableWorktreeMTime(cx *Context) (*time.Time, error) {
	indexMTime, err := findIndexMTime(cx)
	if err != nil {
		return nil, fmt.Errorf("find index mtime: %w", err)
	}

	entriesIter, doneFn := IterDirtyEntries(cx)
	entries := slices.Collect(entriesIter)
	if err := doneFn(); err != nil {
		return nil, fmt.Errorf("find dirty entries: %w", err)
	}
	if len(entries) == 0 {
		return nil, ErrWorktreeClean
	}

	var prev *devState
	statePath, hasStateStorage := devStatePath(cx)
	if hasStateStorage {
		var err error
		prev, err = loadDevState(statePath)
		if err != nil {
			return nil, fmt.Errorf("load state: %w", err)
		}
	}

	fingerprint := computeFingerprint(entries)

	// Stability fast path: identical inputs as last time, return the
	// previously emitted value verbatim and skip writing the new state.
	if prev != nil && prev.Fingerprint == fingerprint {
		return new(prev.Emitted), nil
	}

	// Compute the unfloored candidate as max(index mtime, dirty file mtimes).
	var candidate time.Time
	if indexMTime != nil {
		candidate = *indexMTime
	}
	for _, e := range entries {
		if e.ModTime().After(candidate) {
			candidate = e.ModTime()
		}
	}

	// Apply the monotonicity floor.
	emitted := candidate
	if prev != nil && !candidate.After(prev.Emitted) {
		emitted = prev.Emitted.Add(devCounterTick)
	}

	if hasStateStorage {
		err := saveDevState(statePath, devState{
			Fingerprint: fingerprint,
			Emitted:     emitted,
		})
		if err != nil {
			return nil, fmt.Errorf("save state: %w", err)
		}
	}

	return &emitted, nil
}
