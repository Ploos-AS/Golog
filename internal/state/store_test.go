package state

import (
	"path/filepath"
	"testing"
)

func TestFileStorePersistsAcrossOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("counter", "41"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("user.alice.lang", "no"); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := reopened.Get("counter"); !ok || got != "41" {
		t.Fatalf("counter = %q, %v", got, ok)
	}
	if got, ok := reopened.Get("user.alice.lang"); !ok || got != "no" {
		t.Fatalf("lang = %q, %v", got, ok)
	}

	if err := reopened.Delete("counter"); err != nil {
		t.Fatal(err)
	}
	third, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := third.Get("counter"); ok {
		t.Fatal("deleted key survived reopen")
	}
}

func TestFlattenDeterministicAndEscaped(t *testing.T) {
	got := Flatten(map[string]string{"b": "x;y", "a": "1=2"})
	if got != `a=1\=2;b=x\;y` {
		t.Fatalf("got %q", got)
	}
}
