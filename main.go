package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/shuymn/pommitlint/internal/cli"
)

func main() {
	exitCode, err := cli.Run(context.Background(), &cli.Options{
		Args:    os.Args[1:],
		Stdin:   nil,
		Stdout:  nil,
		Stderr:  nil,
		WorkDir: "",
	})
	if err != nil && !errors.Is(err, cli.ErrLintFailed) {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(exitCode)
}
