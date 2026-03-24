package lint_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

func TestParserSplitsBodyAndFooter(t *testing.T) {
	t.Parallel()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	message, err := lint.Parse("feat(parser): add replay coverage\n\nbody line\n\nRefs: #123", &schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if diff := cmp.Diff([]string{"body line"}, message.BodyLines); diff != "" {
		t.Fatalf("BodyLines mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]string{"Refs: #123"}, message.FooterLines); diff != "" {
		t.Fatalf("FooterLines mismatch (-want +got):\n%s", diff)
	}

	if !message.BodyLeadingBlank {
		t.Fatal("BodyLeadingBlank = false, want true")
	}

	if !message.FooterLeadingBlank {
		t.Fatal("FooterLeadingBlank = false, want true")
	}
}

func TestParserKeepsHeaderSeparatorForFooterOnlyMessage(t *testing.T) {
	t.Parallel()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	message, err := lint.Parse("feat(parser): add replay coverage\n\nRefs: #123", &schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(message.BodyLines) != 0 {
		t.Fatalf("BodyLines = %#v, want empty", message.BodyLines)
	}

	if diff := cmp.Diff([]string{"Refs: #123"}, message.FooterLines); diff != "" {
		t.Fatalf("FooterLines mismatch (-want +got):\n%s", diff)
	}

	if !message.FooterLeadingBlank {
		t.Fatal("FooterLeadingBlank = false, want true")
	}
}
