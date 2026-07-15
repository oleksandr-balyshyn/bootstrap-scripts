package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stateVersion is bumped when the on-disk state schema changes in an
// incompatible way. Older files are ignored rather than misread.
const stateVersion = 1

// State records which commands have already completed successfully so an
// interrupted run can resume without re-executing finished work.
//
// Entries are keyed by a content hash of the command (module, step, and the
// rendered command line). Changing a command in config produces a new key, so
// edited steps run again while untouched steps are skipped.
type State struct {
	path      string
	Version   int                   `json:"version"`
	Profile   string                `json:"profile,omitempty"`
	UpdatedAt string                `json:"updatedAt,omitempty"`
	Completed map[string]stateEntry `json:"completed"`
}

type stateEntry struct {
	Module      string `json:"module"`
	Step        string `json:"step"`
	Command     string `json:"command"`
	CompletedAt string `json:"completedAt"`
}

// LoadState reads the state file at path. A missing file yields an empty state.
// A file written by an incompatible schema version is discarded and replaced by
// an empty state so a stale format never blocks or corrupts a run.
func LoadState(path string) (*State, error) {
	state := &State{
		path:      path,
		Version:   stateVersion,
		Completed: map[string]stateEntry{},
	}
	if path == "" {
		return state, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}

	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("parse state file %s: %w", path, err)
	}
	if loaded.Version != stateVersion || loaded.Completed == nil {
		return state, nil
	}
	loaded.path = path
	loaded.Version = stateVersion
	state = &loaded
	return state, nil
}

// commandKey is the stable identity of a command within the plan. It is derived
// from the command content so an unchanged command is recognised across runs.
func commandKey(moduleID, stepName string, command Command) string {
	sum := sha256.Sum256([]byte(moduleID + "\x00" + stepName + "\x00" + command.String()))
	return hex.EncodeToString(sum[:])
}

// commandPreview renders a compact, single-line command summary for the state
// file. The full command content is captured by the key hash; this field is
// only for humans reading the state file, so long shell scripts are truncated.
func commandPreview(command Command) string {
	preview := strings.Join(strings.Fields(command.String()), " ")
	const max = 160
	if len(preview) > max {
		preview = preview[:max] + "…"
	}
	return preview
}

// Done reports whether the command has already completed successfully.
func (s *State) Done(key string) bool {
	if s == nil {
		return false
	}
	_, ok := s.Completed[key]
	return ok
}

// Count returns the number of recorded completions.
func (s *State) Count() int {
	if s == nil {
		return 0
	}
	return len(s.Completed)
}

// MarkDone records a successful command and flushes the state to disk so an
// interruption immediately after this call still preserves the progress.
func (s *State) MarkDone(key, moduleID, stepName string, command Command, now time.Time) error {
	if s == nil {
		return nil
	}
	if s.Completed == nil {
		s.Completed = map[string]stateEntry{}
	}
	s.Completed[key] = stateEntry{
		Module:      moduleID,
		Step:        stepName,
		Command:     commandPreview(command),
		CompletedAt: now.UTC().Format(time.RFC3339),
	}
	s.UpdatedAt = now.UTC().Format(time.RFC3339)
	return s.save()
}

// save writes the state atomically (temp file + rename) so a crash mid-write
// never leaves a truncated, unparseable state file.
func (s *State) save() error {
	if s == nil || s.path == "" {
		return nil
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
