package rules

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "10-rules.pl"), []byte(`loaded(ok).`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverRegistry(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "base"), `{
		"id":"base","version":"1.0.0","roles":["core"],
		"commands":["!help"],"capabilities":["core.help"]
	}`)
	writeManifest(t, filepath.Join(root, "amiga"), `{
		"id":"amiga","version":"0.1.0","description":"Amiga expert rules",
		"roles":["amiga"],"commands":["!amiga"],
		"capabilities":["expert.amiga"],"depends":["base"]
	}`)

	registry, err := DiscoverRegistry([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := registry.IDs(), []string{"amiga", "base"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs() = %#v, want %#v", got, want)
	}
	if got, want := registry.Roles(), []string{"amiga", "core"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Roles() = %#v, want %#v", got, want)
	}
	if got, want := registry.Capabilities(), []string{"core.help", "expert.amiga"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Capabilities() = %#v, want %#v", got, want)
	}
	if pack, ok := registry.Lookup("AMIGA"); !ok || pack.Version != "0.1.0" {
		t.Fatalf("Lookup(amiga) = %#v, %v", pack, ok)
	}
}

func TestDiscoverRegistryRejectsMissingDependency(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "amiga"), `{
		"id":"amiga","version":"0.1.0","depends":["core"]
	}`)
	if _, err := DiscoverRegistry([]string{root}); err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestDiscoverRegistryAllowsLegacyRuleFileWithoutManifest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "legacy.pl")
	if err := os.WriteFile(path, []byte(`legacy(ok).`), 0o644); err != nil {
		t.Fatal(err)
	}
	registry, err := DiscoverRegistry([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if registry.Len() != 0 {
		t.Fatalf("registry len = %d, want 0", registry.Len())
	}
}
