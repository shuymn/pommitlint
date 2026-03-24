package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	//nolint:forbidigo // Process-boundary entrypoints own the root context.
	ctx := context.Background()

	if err := syncPreset(ctx, syncOptions{
		ArtifactPath: defaultArtifactPath(),
		Resolve:      resolveWithBun,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "sync preset: %v\n", err)
		os.Exit(1)
	}
}

func defaultArtifactPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("internal", "preset", "preset.json")
	}

	return filepath.Join(filepath.Dir(file), "..", "..", "internal", "preset", "preset.json")
}
