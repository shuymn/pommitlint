package lint_test

import (
	"testing"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

const benchMessage = "feat(parser): add replay coverage\n\n" +
	"This commit introduces replay coverage for the parser module.\n" +
	"It covers all edge cases for body and footer parsing.\n\n" +
	"Refs: #123\n" +
	"Co-authored-by: Alice <alice@example.com>"

func BenchmarkLint(b *testing.B) {
	schema, err := preset.Load()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := lint.Lint(benchMessage, "stdin", &schema); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	schema, err := preset.Load()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := lint.Parse(benchMessage, &schema); err != nil {
			b.Fatal(err)
		}
	}
}
