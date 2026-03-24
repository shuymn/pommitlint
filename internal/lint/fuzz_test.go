package lint_test

import (
	"testing"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

func FuzzLint(f *testing.F) {
	schema, err := preset.Load()
	if err != nil {
		f.Fatalf("Load() error = %v", err)
	}

	f.Add("feat: add parser")
	f.Add("feat(parser): add parser\n\nbody\n\nRefs: #123")
	f.Add("revert: `$(rm -rf /)`\n\nhttps://example.com/" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	f.Add("")

	f.Fuzz(func(_ *testing.T, message string) {
		// Only detect panics/crashes; validation errors are expected for arbitrary input.
		_, _ = lint.Lint(message, "fuzz", &schema) //nolint:errcheck // fuzz: only detect panics; errors are expected for arbitrary input
	})
}
