package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/shuymn/pommitlint/internal/cli"
)

func TestCLIStdinReplay(t *testing.T) {
	t.Parallel()

	result := runLint(t, &runInput{
		stdin: "feat(parser): add replay coverage\n\nhttps://example.com/" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n" +
			"wrap this line because it should fail the body length rule once it is long enough to exceed one hundred characters\n\nRefs: #123",
		format: "text",
	})

	if result.exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", result.exitCode)
	}

	if diff := cmp.Diff([]string{
		"input: stdin",
		"status: invalid",
		"errors: 1",
		"warnings: 0",
		"",
		"error: body-max-line-length body line 2 exceeds max length 100",
	}, splitLines(result.stdout)); diff != "" {
		t.Fatalf("stdout mismatch (-want +got):\n%s", diff)
	}

	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty", result.stderr)
	}
}

func TestCLIMessageReplay(t *testing.T) {
	t.Parallel()

	result := runLint(t, &runInput{
		message: "feat(parser): add replay coverage",
		format:  "json",
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", result.exitCode)
	}

	var got cli.ExportJSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	want := cli.ExportJSONReport{
		Source:       "message",
		Valid:        true,
		ErrorCount:   0,
		WarningCount: 0,
		Findings:     []cli.ExportJSONFinding{},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("report mismatch (-want +got):\n%s", diff)
	}
}

func TestCLIEditReplay(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	editPath := filepath.Join(tempDir, "COMMIT_EDITMSG")
	message := "feat: Add parser\n\nbody\n"
	if err := os.WriteFile(editPath, []byte(message), 0o600); err != nil {
		t.Fatalf("write edit file: %v", err)
	}

	result := runLint(t, &runInput{
		editPath: editPath,
		format:   "json",
	})

	if result.exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", result.exitCode)
	}

	var got cli.ExportJSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	want := cli.ExportJSONReport{
		Source:       "edit",
		Valid:        false,
		ErrorCount:   1,
		WarningCount: 0,
		Findings: []cli.ExportJSONFinding{
			{
				Rule:    "subject-case",
				Level:   "error",
				Field:   "subject",
				Message: "subject must not be sentence-case, start-case, pascal-case, or upper-case",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("report mismatch (-want +got):\n%s", diff)
	}
}

func TestCLIFileReplay(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	messagePath := filepath.Join(tempDir, "COMMIT_MSG")
	message := "feat: add parser\nbody without separator\n"
	if err := os.WriteFile(messagePath, []byte(message), 0o600); err != nil {
		t.Fatalf("write message file: %v", err)
	}

	result := runLint(t, &runInput{
		filePath: messagePath,
		format:   "json",
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", result.exitCode)
	}

	var got cli.ExportJSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	if got.WarningCount != 1 {
		t.Fatalf("WarningCount = %d, want 1", got.WarningCount)
	}

	if got.ErrorCount != 0 {
		t.Fatalf("ErrorCount = %d, want 0", got.ErrorCount)
	}

	if len(got.Findings) == 0 {
		t.Fatal("Findings is empty, expected at least one finding")
	}

	if diff := cmp.Diff(cli.ExportJSONFinding{
		Rule:    "body-leading-blank",
		Level:   "warning",
		Field:   "body",
		Message: "body must begin with a blank line",
	}, got.Findings[0]); diff != "" {
		t.Fatalf("finding mismatch (-want +got):\n%s", diff)
	}
}

type runInput struct {
	stdin    string
	message  string
	filePath string
	editPath string
	format   string
}

type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runLint(t *testing.T, input *runInput) runResult {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	options := cli.Options{
		Stdin:  bytes.NewBufferString(input.stdin),
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"lint", "--format", input.format},
	}

	if input.message != "" {
		options.Args = append(options.Args, "--message", input.message)
	}

	if input.filePath != "" {
		options.Args = append(options.Args, "--file", input.filePath)
	}

	if input.editPath != "" {
		options.Args = append(options.Args, "--edit", input.editPath)
	}

	exitCode, err := cli.Run(t.Context(), &options)
	if err != nil && !errors.Is(err, cli.ErrLintFailed) {
		t.Fatalf("Run() error = %v", err)
	}

	return runResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
}

func splitLines(raw string) []string {
	return strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
}
