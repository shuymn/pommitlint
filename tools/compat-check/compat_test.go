//go:build compat

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/shuymn/pommitlint/internal/cli"
	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

type cliJSONReport struct {
	Source  string `json:"source"`
	Valid   bool   `json:"valid"`
	Ignored bool   `json:"ignored"`
}

func TestCompatDifferential(t *testing.T) {
	t.Parallel()

	corpus := loadCorpus(t)
	clResults := runCommitlint(t)
	schema := loadSchema(t)

	clByID := indexByID(clResults)

	var diffs []comparisonDiff

	for _, entry := range corpus {
		if slices.Contains(entry.Tags, "ignored") || slices.Contains(entry.Tags, "not-ignored") {
			continue
		}

		cl, ok := clByID[entry.ID]
		if !ok {
			t.Errorf("corpus entry %q not found in commitlint results", entry.ID)
			continue
		}

		result, err := lint.Lint(entry.Message, "compat", &schema)
		if err != nil {
			t.Errorf("pommitlint lint error for %q: %v", entry.ID, err)
			continue
		}

		pl := tolintResult(entry.ID, result)

		if diff := Compare(cl, pl, entry.Message); diff != nil {
			diffs = append(diffs, *diff)
		}
	}

	if len(diffs) > 0 {
		var sb strings.Builder
		fmt.Fprintf(&sb, "found %d compatibility difference(s):\n\n", len(diffs))
		for _, d := range diffs {
			fmt.Fprintf(&sb, "  [%s] message=%q\n", d.ID, d.Message)
			fmt.Fprintf(&sb, "    valid:   commitlint=%v pommitlint=%v\n", d.CommitlintValid, d.PommitlintValid)
			fmt.Fprintf(&sb, "    ignored: commitlint=%v pommitlint=%v\n", d.CommitlintIgnored, d.PommitlintIgnored)
			if len(d.OnlyCommitlint) > 0 {
				fmt.Fprintf(&sb, "    only commitlint: %s\n", formatFindings(d.OnlyCommitlint))
			}
			if len(d.OnlyPommitlint) > 0 {
				fmt.Fprintf(&sb, "    only pommitlint: %s\n", formatFindings(d.OnlyPommitlint))
			}
			sb.WriteString("\n")
		}
		t.Error(sb.String())
	}
}

func TestCompatIgnored(t *testing.T) {
	t.Parallel()

	corpus := loadCorpus(t)
	clResults := runCommitlint(t)

	clByID := indexByID(clResults)

	var diffs []string

	for _, entry := range corpus {
		if !slices.Contains(entry.Tags, "ignored") && !slices.Contains(entry.Tags, "not-ignored") {
			continue
		}

		cl, ok := clByID[entry.ID]
		if !ok {
			t.Errorf("corpus entry %q not found in commitlint results", entry.ID)
			continue
		}

		plIgnored, plValid := runPommitlintCLI(t, entry.Message)

		if cl.Ignored != plIgnored {
			diffs = append(diffs, fmt.Sprintf(
				"[%s] ignored: commitlint=%v pommitlint=%v (message=%q)",
				entry.ID, cl.Ignored, plIgnored, entry.Message,
			))
		}

		if cl.Valid != plValid {
			diffs = append(diffs, fmt.Sprintf(
				"[%s] valid: commitlint=%v pommitlint=%v (message=%q)",
				entry.ID, cl.Valid, plValid, entry.Message,
			))
		}
	}

	if len(diffs) > 0 {
		t.Errorf("found %d ignored compatibility difference(s):\n  %s",
			len(diffs), strings.Join(diffs, "\n  "))
	}
}

func loadCorpus(t *testing.T) []CorpusEntry {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(toolDir(), "corpus.json"))
	if err != nil {
		t.Fatalf("read corpus.json: %v", err)
	}

	var corpus []CorpusEntry
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("parse corpus.json: %v", err)
	}

	return corpus
}

func runCommitlint(t *testing.T) []lintResult {
	t.Helper()

	dir := toolDir()
	ctx := t.Context()

	cmd := exec.CommandContext(ctx, "bun", "run", "lint-corpus.ts")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("bun lint-corpus.ts failed: %v\nstderr: %s", err, stderr.String())
	}

	var results []lintResult
	if err := json.Unmarshal(output, &results); err != nil {
		t.Fatalf("parse commitlint results: %v", err)
	}

	return results
}

func loadSchema(t *testing.T) preset.Schema {
	t.Helper()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("preset.Load() error = %v", err)
	}

	return schema
}

func tolintResult(id string, result lint.Result) lintResult {
	findings := make([]findingSummary, 0, len(result.Findings))
	for _, f := range result.Findings {
		findings = append(findings, findingSummary{
			Rule:  string(f.Rule),
			Level: string(f.Level),
		})
	}

	slices.SortFunc(findings, cmpFinding)

	return lintResult{
		ID:       id,
		Valid:    result.Valid,
		Ignored:  result.Ignored,
		Findings: findings,
	}
}

func runPommitlintCLI(t *testing.T, message string) (ignored, valid bool) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	options := cli.Options{
		Args:   []string{"lint", "--message", message, "--format", "json"},
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	_, err := cli.Run(t.Context(), &options)
	if err != nil && !errors.Is(err, cli.ErrLintFailed) {
		t.Fatalf("cli.Run unexpected error: %v", err)
	}

	var report cliJSONReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parse pommitlint CLI output: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return report.Ignored, report.Valid
}

func indexByID(results []lintResult) map[string]lintResult {
	m := make(map[string]lintResult, len(results))
	for _, r := range results {
		m[r.ID] = r
	}
	return m
}

func formatFindings(findings []findingSummary) string {
	parts := make([]string, len(findings))
	for i, f := range findings {
		parts[i] = f.Level + ":" + f.Rule
	}
	return strings.Join(parts, ", ")
}

func toolDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}

	return filepath.Dir(file)
}
