package cli

import (
	"bytes"
	"testing"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

func TestReportTextIncludesCounts(t *testing.T) {
	t.Parallel()

	got := formatTextReport(&lint.Result{
		Source: "stdin",
		Valid:  false,
		Findings: []lint.Finding{
			{
				Rule:    preset.RuleTypeEmpty,
				Level:   lint.LevelError,
				Field:   "type",
				Message: "type may not be empty",
			},
		},
	})

	want := "input: stdin\nstatus: invalid\nerrors: 1\nwarnings: 0\n\nerror: type-empty type may not be empty\n"
	if got != want {
		t.Fatalf("formatTextReport() = %q, want %q", got, want)
	}
}

func TestReportJSONEscapesHTML(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	result := lint.Result{
		Source: "message",
		Valid:  false,
		Findings: []lint.Finding{
			{
				Rule:    preset.RuleSubjectFullStop,
				Level:   lint.LevelError,
				Field:   "subject",
				Message: "<script>",
			},
		},
	}

	if err := writeJSONReport(&buffer, &result); err != nil {
		t.Fatalf("writeJSONReport() error = %v", err)
	}

	if bytes.Contains(buffer.Bytes(), []byte("<script>")) {
		t.Fatal("JSON report did not escape HTML content")
	}
}
