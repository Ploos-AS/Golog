package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gprolog "github.com/Ploos-AS/Golog/internal/prolog"
	"github.com/Ploos-AS/Golog/internal/prolog/ichiban"
)

func TestDiscoveryCommandsReflectLoadedRegistry(t *testing.T) {
	root := t.TempDir()
	pack := filepath.Join(root, "amiga")
	if err := os.MkdirAll(pack, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "id": "amiga",
  "version": "0.1.0",
  "description": "Amiga expert rules",
  "roles": ["amiga"],
  "commands": ["!amiga"],
  "capabilities": ["expert.amiga"],
  "depends": []
}`
	if err := os.WriteFile(filepath.Join(pack, ManifestName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, "10-rules.pl"), []byte(`on_privmsg(_Nick, _Target, "!amiga", "Amiga ready").`), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, _, err := Load([]string{pack}, func() gprolog.Engine { return ichiban.New() })
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		command string
		want    []string
	}{
		{"!help", []string{"!amiga", "!help", "!packs", "!experts", "!capabilities"}},
		{"!packs", []string{"packs: amiga"}},
		{"!packs amiga", []string{"pack amiga 0.1.0", "roles=amiga", "commands=!amiga", "capabilities=expert.amiga"}},
		{"!experts", []string{"experts: amiga"}},
		{"!capabilities", []string{"capabilities: expert.amiga"}},
	}

	for _, tc := range cases {
		reply, ok, err := engine.QueryReply(`on_privmsg(?, ?, ?, Reply).`, "alice", "#golog", tc.command)
		if err != nil {
			t.Fatalf("%s: %v", tc.command, err)
		}
		if !ok {
			t.Fatalf("%s: no reply", tc.command)
		}
		for _, want := range tc.want {
			if !strings.Contains(reply, want) {
				t.Fatalf("%s: reply %q does not contain %q", tc.command, reply, want)
			}
		}
	}
}
