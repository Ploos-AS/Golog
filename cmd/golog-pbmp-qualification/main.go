package main

import (
	"fmt"
	"os"

	"github.com/Ploos-AS/Golog/internal/pbmp"
)

func main() {
	path := os.Getenv("GOLOG_PBMP_SOCKET")
	if path == "" {
		fmt.Fprintln(os.Stderr, "GOLOG_PBMP_SOCKET is required")
		os.Exit(2)
	}
	if err := pbmp.Serve(path, pbmp.State{Nick: "golog-qualification", Network: "irc.example.invalid"}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
