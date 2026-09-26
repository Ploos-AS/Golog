package rules

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const ManifestName = "pack.json"

var (
	packIDPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	versionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
)

type Manifest struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Description  string   `json:"description,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	Commands     []string `json:"commands,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Depends      []string `json:"depends,omitempty"`
	Path         string   `json:"-"`
}

type Registry struct {
	packs map[string]Manifest
	order []string
}

func EmptyRegistry() *Registry {
	return &Registry{packs: map[string]Manifest{}}
}

// DiscoverRegistry finds pack.json manifests associated with configured rule
// paths, validates them, and returns a deterministic registry. Plain .pl files
// do not require a manifest and remain fully supported.
func DiscoverRegistry(paths []string) (*Registry, error) {
	manifestPaths, err := discoverManifestPaths(paths)
	if err != nil {
		return nil, err
	}

	registry := EmptyRegistry()
	for _, path := range manifestPaths {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read pack manifest %q: %w", path, err)
		}
		var manifest Manifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			return nil, fmt.Errorf("decode pack manifest %q: %w", path, err)
		}
		manifest.Path = filepath.Clean(path)
		normalizeManifest(&manifest)
		if err := validateManifest(manifest); err != nil {
			return nil, fmt.Errorf("pack manifest %q: %w", path, err)
		}
		if old, exists := registry.packs[manifest.ID]; exists {
			return nil, fmt.Errorf("duplicate pack id %q in %q and %q", manifest.ID, old.Path, manifest.Path)
		}
		registry.packs[manifest.ID] = manifest
		registry.order = append(registry.order, manifest.ID)
	}
	sort.Strings(registry.order)

	for _, id := range registry.order {
		manifest := registry.packs[id]
		for _, dependency := range manifest.Depends {
			if _, ok := registry.packs[dependency]; !ok {
				return nil, fmt.Errorf("pack %q depends on unloaded pack %q", id, dependency)
			}
		}
	}
	return registry, nil
}

func discoverManifestPaths(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var manifests []string
	add := func(path string) {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		manifests = append(manifests, path)
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
			candidate := filepath.Join(filepath.Dir(path), ManifestName)
			if _, err := os.Stat(candidate); err == nil {
				add(candidate)
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("stat pack manifest %q: %w", candidate, err)
			}
			continue
		}

		err = filepath.WalkDir(path, func(entry string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == ManifestName {
				add(entry)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk manifests under %q: %w", path, err)
		}
	}
	sort.Strings(manifests)
	return manifests, nil
}

func normalizeManifest(manifest *Manifest) {
	manifest.ID = strings.ToLower(strings.TrimSpace(manifest.ID))
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.Roles = normalizeList(manifest.Roles, true)
	manifest.Commands = normalizeList(manifest.Commands, false)
	manifest.Capabilities = normalizeList(manifest.Capabilities, true)
	manifest.Depends = normalizeList(manifest.Depends, true)
}

func normalizeList(values []string, lower bool) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if lower {
			value = strings.ToLower(value)
		}
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func validateManifest(manifest Manifest) error {
	if !packIDPattern.MatchString(manifest.ID) {
		return fmt.Errorf("invalid id %q", manifest.ID)
	}
	if !versionPattern.MatchString(manifest.Version) {
		return fmt.Errorf("invalid version %q; expected semver-like x.y.z", manifest.Version)
	}
	for _, dependency := range manifest.Depends {
		if !packIDPattern.MatchString(dependency) {
			return fmt.Errorf("invalid dependency id %q", dependency)
		}
		if dependency == manifest.ID {
			return fmt.Errorf("pack %q cannot depend on itself", manifest.ID)
		}
	}
	return nil
}

func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.order)
}

func (r *Registry) Packs() []Manifest {
	if r == nil {
		return nil
	}
	out := make([]Manifest, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.packs[id])
	}
	return out
}

func (r *Registry) IDs() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.order...)
}

func (r *Registry) Lookup(id string) (Manifest, bool) {
	if r == nil {
		return Manifest{}, false
	}
	manifest, ok := r.packs[strings.ToLower(strings.TrimSpace(id))]
	return manifest, ok
}

func (r *Registry) Roles() []string        { return r.collect(func(m Manifest) []string { return m.Roles }) }
func (r *Registry) Commands() []string     { return r.collect(func(m Manifest) []string { return m.Commands }) }
func (r *Registry) Capabilities() []string { return r.collect(func(m Manifest) []string { return m.Capabilities }) }

func (r *Registry) collect(values func(Manifest) []string) []string {
	if r == nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, id := range r.order {
		for _, value := range values(r.packs[id]) {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
