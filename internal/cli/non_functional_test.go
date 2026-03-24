package cli_test

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/shuymn/pommitlint/internal/cli"
)

func TestNoNetworkLint(t *testing.T) {
	t.Parallel()

	result := runCommand(t, &commandInput{
		args: []string{"lint", "--message", "feat: stay offline", "--format", "json"},
	})
	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", result.exitCode, result.stderr)
	}

	repoRoot := repoRootFromThisFile(t)
	runtimeDirs := []string{
		filepath.Join(repoRoot, "internal", "lint"),
		filepath.Join(repoRoot, "internal", "preset"),
	}
	for _, dir := range runtimeDirs {
		assertNoForbiddenImports(t, dir, []string{"net", "os/exec"})
	}
}

func TestNoShellInterpolation(t *testing.T) {
	t.Parallel()

	marker := filepath.Join(t.TempDir(), "executed-from-message")
	message := "feat: $(touch " + marker + ")"

	result := runCommand(t, &commandInput{
		args: []string{"lint", "--message", message, "--format", "json"},
	})
	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", result.exitCode, result.stderr)
	}

	_, err := os.Stat(marker)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("message content created side effect, stat error = %v", err)
	}
}

func TestBoundedInputBehavior(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	var stdout strings.Builder
	var stderr strings.Builder
	oversizedLine := strings.Repeat("a", 256*1024)
	message := "feat: keep runtime bounded\n\n" + oversizedLine + "\n\nRefs: #123"

	exitCode, err := cli.Run(ctx, &cli.Options{
		Args:   []string{"lint", "--message", message, "--format", "text"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil && !errors.Is(err, cli.ErrLintFailed) {
		t.Fatalf("Run() error = %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stderr=%q", exitCode, stderr.String())
	}
	if ctx.Err() != nil {
		t.Fatalf("context expired before lint completed: %v", ctx.Err())
	}

	output := stdout.String()
	if !strings.Contains(output, "error: body-max-line-length") {
		t.Fatalf("stdout = %q, want body-max-line-length finding", output)
	}
	const maxOutputLines = 8 // bounded output: body-max-line-length finding + header lines
	if strings.Count(output, "\n") > maxOutputLines {
		t.Fatalf("stdout emitted too many lines for oversized input: %q", output)
	}
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func assertNoForbiddenImports(t *testing.T, dir string, forbiddenPrefixes []string) {
	t.Helper()

	fileSet := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.HasSuffix(d.Name(), "_test.go") || filepath.Ext(d.Name()) != ".go" {
			return nil
		}
		parsed, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s) error = %v", path, err)
		}
		for _, imported := range parsed.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			for _, prefix := range forbiddenPrefixes {
				if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
					t.Fatalf("%s imports forbidden package %q", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s) error = %v", dir, err)
	}
}
