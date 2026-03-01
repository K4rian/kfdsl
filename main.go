package main

import (
	"fmt"
	"os"

	"github.com/K4rian/kfdsl/internal/launcher"
)

func main() {
	if err := launcher.New().Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
