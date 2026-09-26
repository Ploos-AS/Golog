package ichiban

import "testing"

func TestLoadAndAsk(t *testing.T) {
	engine := New()

	if err := engine.Load(`hello("golog").`); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	ok, err := engine.Ask(`hello(?).`, "golog")
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if !ok {
		t.Fatal("Ask() = false, want true")
	}

	ok, err = engine.Ask(`hello(?).`, "someone_else")
	if err != nil {
		t.Fatalf("Ask() negative query error = %v", err)
	}
	if ok {
		t.Fatal("Ask() = true for unknown fact, want false")
	}
}
