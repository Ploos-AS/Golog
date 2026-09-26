package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store is the persistence contract used by the rule worker.
type Store interface {
	Get(key string) (string, bool)
	Set(key, value string) error
	Delete(key string) error
	Snapshot() map[string]string
}

// FileStore persists string key/value state as JSON.
type FileStore struct {
	mu   sync.RWMutex
	path string
	data map[string]string
}

func Open(path string) (*FileStore, error) {
	s := &FileStore{path: path, data: map[string]string{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	if len(b) != 0 {
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, err
		}
	}
	if s.data == nil {
		s.data = map[string]string{}
	}
	return s, nil
}

func (s *FileStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *FileStore) Set(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("state key must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return s.persistLocked()
}

func (s *FileStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return s.persistLocked()
}

func (s *FileStore) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

func (s *FileStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil && filepath.Dir(s.path) != "." {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Flatten returns a deterministic escaped representation for Prolog.
func Flatten(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, escape(k)+"="+escape(values[k]))
	}
	return strings.Join(parts, ";")
}

func escape(s string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", "=", "\\=")
	return r.Replace(s)
}
