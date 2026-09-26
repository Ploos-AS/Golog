package rules

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
)

type EngineFactory func() gprolog.Engine

// Expand resolves rule files and directories into a stable, de-duplicated list
// of .pl files. Directories are walked recursively and sorted lexicographically.
func Expand(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var files []string

	add := func(path string) {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}

	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat rule path %q: %w", path, err)
		}
		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(path), ".pl") {
				add(path)
			}
			continue
		}

		err = filepath.WalkDir(path, func(entry string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(entry), ".pl") {
				return nil
			}
			add(entry)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk rule path %q: %w", path, err)
		}
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no Prolog rule files found")
	}
	return files, nil
}

// Load creates a fresh engine and loads all resolved rule files into it.
// Callers can therefore validate a complete rule pack set before swapping it
// into a running worker.
func Load(paths []string, factory EngineFactory) (gprolog.Engine, []string, error) {
	if factory == nil {
		return nil, nil, fmt.Errorf("engine factory must not be nil")
	}
	files, err := Expand(paths)
	if err != nil {
		return nil, nil, err
	}
	engine := factory()
	if engine == nil {
		return nil, nil, fmt.Errorf("engine factory returned nil")
	}
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read rule file %q: %w", path, err)
		}
		if err := engine.Load(string(source)); err != nil {
			return nil, nil, fmt.Errorf("load rule file %q: %w", path, err)
		}
	}
	return engine, files, nil
}
