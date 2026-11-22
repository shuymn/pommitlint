package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "compat-check is a test-only tool. Run: mise run compat-check")
	os.Exit(1)
}
