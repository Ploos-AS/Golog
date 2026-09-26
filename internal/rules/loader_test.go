package rules

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func TestExpandDirectoryIsRecursiveAndStable(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(name, body string) string {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return filepath.Clean(path)
	}

	b := mustWrite("pack/20-b.pl", `b("ok").`)
	a := mustWrite("pack/10-a.pl", `a("ok").`)
	c := mustWrite("pack/sub/30-c.pl", `c("ok").`)
	_ = mustWrite("pack/README.txt", "ignored")

	got, err := Expand([]string{filepath.Join(root, "pack")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{a, b, c}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expand() = %#v, want %#v", got, want)
	}
}

func TestLoadCombinesMultipleRuleFiles(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"10-base.pl": `expert(irc).`,
		"20-extra.pl": `expert(amiga).`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	engine, files, err := Load([]string{root}, func() gprolog.Engine { return ichiban.New() })
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("loaded files = %d, want 2", len(files))
	}
	for _, query := range []string{`expert(irc).`, `expert(amiga).`} {
		ok, err := engine.Ask(query)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("%s not loaded", query)
		}
	}
}
