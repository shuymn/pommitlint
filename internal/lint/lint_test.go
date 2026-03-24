package lint_test

import (
	"testing"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

func TestLintIgnoresURLInBodyLineLength(t *testing.T) {
	t.Parallel()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	message := "feat(parser): add replay coverage\n\nhttps://example.com/" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n" +
		"Refs: #123"

	result, err := lint.Lint(message, "stdin", &schema)
	if err != nil {
		t.Fatalf("Lint() error = %v", err)
	}

	if !result.Valid {
		t.Fatalf("Valid = %v, want true", result.Valid)
	}
}

func TestLintSubjectCaseSkipsNonLatin(t *testing.T) {
	t.Parallel()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	result, err := lint.Lint("feat: 追加する", "message", &schema)
	if err != nil {
		t.Fatalf("Lint() error = %v", err)
	}

	for _, finding := range result.Findings {
		if finding.Rule == preset.RuleSubjectCase {
			t.Fatal("subject-case finding for non-Latin subject")
		}
	}
}

func TestLintFooterMaxLineLengthDoesNotIgnoreURL(t *testing.T) {
	t.Parallel()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	message := "feat(parser): add replay coverage\n\nbody\n\nRefs: https://example.com/" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	result, err := lint.Lint(message, "stdin", &schema)
	if err != nil {
		t.Fatalf("Lint() error = %v", err)
	}

	found := false
	for _, finding := range result.Findings {
		if finding.Rule == preset.RuleFooterMaxLineLength {
			found = true
		}
	}

	if !found {
		t.Fatal("missing footer-max-line-length finding")
	}
}
