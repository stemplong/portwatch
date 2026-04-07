// Package state manages the persistence of observed port states between
// scanner runs, enabling portwatch to detect changes over time.
package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// PortEntry represents a single observed open port with metadata.
type PortEntry struct {
	Protocol  string    `json:"protocol"`
	LocalAddr string    `json:"local_addr"`
	LocalPort uint16    `json:"local_port"`
	PID       int       `json:"pid,omitempty"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// Snapshot holds the full set of ports observed during a single scan.
type Snapshot struct {
	Ports     []PortEntry `json:"ports"`
	ScannedAt time.Time   `json:"scanned_at"`
}

// Diff describes the changes between two consecutive snapshots.
type Diff struct {
	Opened []PortEntry
	Closed []PortEntry
}

// HasChanges returns true if any ports were opened or closed.
func (d *Diff) HasChanges() bool {
	return len(d.Opened) > 0 || len(d.Closed) > 0
}

// Store persists port state to disk and tracks changes between scans.
type Store struct {
	mu       sync.RWMutex
	filePath string
	current  Snapshot
}

// New creates a Store backed by the given file path. If the file exists,
// the previous snapshot is loaded so diffs can be computed on the first run.
func New(filePath string) (*Store, error) {
	s := &Store{filePath: filePath}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

// Update replaces the current snapshot with the new one, persists it to disk,
// and returns a Diff describing what changed.
func (s *Store) Update(next Snapshot) (Diff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	diff := compare(s.current, next)
	s.current = next

	if err := s.save(); err != nil {
		return diff, err
	}
	return diff, nil
}

// Current returns a copy of the most recently stored snapshot.
func (s *Store) Current() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// load reads the persisted snapshot from disk.
func (s *Store) load() error {
	f, err := os.Open(s.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&s.current)
}

// save writes the current snapshot to disk atomically via a temp file rename.
func (s *Store) save() error {
	tmp := s.filePath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.current); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// compare produces a Diff by comparing previous and next snapshots.
func compare(prev, next Snapshot) Diff {
	prevMap := indexPorts(prev.Ports)
	nextMap := indexPorts(next.Ports)

	var diff Diff
	for key, entry := range nextMap {
		if _, exists := prevMap[key]; !exists {
			diff.Opened = append(diff.Opened, entry)
		}
	}
	for key, entry := range prevMap {
		if _, exists := nextMap[key]; !exists {
			diff.Closed = append(diff.Closed, entry)
		}
	}
	return diff
}

// indexPorts builds a lookup map keyed by "protocol:addr:port".
func indexPorts(ports []PortEntry) map[string]PortEntry {
	m := make(map[string]PortEntry, len(ports))
	for _, p := range ports {
		key := p.Protocol + ":" + p.LocalAddr + ":" + string(rune(p.LocalPort))
		m[key] = p
	}
	return m
}
