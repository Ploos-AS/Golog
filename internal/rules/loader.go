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

// Load creates a fresh engine and loads the generated core discovery clauses
// plus the complete resolved ruleset in one interpreter operation. This
// preserves clauses for predicates spread across multiple pack files and makes
// validation atomic before a hot-reload swap.
func Load(paths []string, factory EngineFactory) (gprolog.Engine, []string, error) {
	if factory == nil {
		return nil, nil, fmt.Errorf("engine factory must not be nil")
	}
	files, err := Expand(paths)
	if err != nil {
		return nil, nil, err
	}
	registry, err := DiscoverRegistry(paths)
	if err != nil {
		return nil, nil, fmt.Errorf("discover rule packs: %w", err)
	}

	var source strings.Builder
	source.WriteString(DiscoverySource(registry))
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("read rule file %q: %w", path, err)
		}
		fmt.Fprintf(&source, "\n%% --- begin %s ---\n", filepath.ToSlash(path))
		source.Write(body)
		if len(body) == 0 || body[len(body)-1] != '\n' {
			source.WriteByte('\n')
		}
		fmt.Fprintf(&source, "%% --- end %s ---\n", filepath.ToSlash(path))
	}

	engine := factory()
	if engine == nil {
		return nil, nil, fmt.Errorf("engine factory returned nil")
	}
	if err := engine.Load(source.String()); err != nil {
		return nil, nil, fmt.Errorf("load combined rule packs: %w", err)
	}
	return engine, files, nil
}
