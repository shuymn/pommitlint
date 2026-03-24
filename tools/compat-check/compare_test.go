package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCompareIdentical(t *testing.T) {
	t.Parallel()

	cl := LintResult{
		ID:    "test",
		Valid: false,
		Findings: []FindingSummary{
			{Rule: "type-enum", Level: "error"},
		},
	}
	pl := LintResult{
		ID:    "test",
		Valid: false,
		Findings: []FindingSummary{
			{Rule: "type-enum", Level: "error"},
		},
	}

	if diff := Compare(cl, pl, "foo: msg"); diff != nil {
		t.Errorf("expected nil diff, got %+v", diff)
	}
}

func TestCompareIdenticalDifferentOrder(t *testing.T) {
	t.Parallel()

	cl := LintResult{
		ID:    "test",
		Valid: false,
		Findings: []FindingSummary{
			{Rule: "type-case", Level: "error"},
			{Rule: "type-enum", Level: "error"},
		},
	}
	pl := LintResult{
		ID:    "test",
		Valid: false,
		Findings: []FindingSummary{
			{Rule: "type-enum", Level: "error"},
			{Rule: "type-case", Level: "error"},
		},
	}

	if diff := Compare(cl, pl, "FIX: msg"); diff != nil {
		t.Errorf("expected nil diff, got %+v", diff)
	}
}

func TestCompareValidMismatch(t *testing.T) {
	t.Parallel()

	cl := LintResult{ID: "test", Valid: true}
	pl := LintResult{ID: "test", Valid: false, Findings: []FindingSummary{
		{Rule: "type-enum", Level: "error"},
	}}

	diff := Compare(cl, pl, "msg")
	if diff == nil {
		t.Fatal("expected diff, got nil")
	}

	if diff.CommitlintValid != true || diff.PommitlintValid != false {
		t.Errorf("valid mismatch: cl=%v, pl=%v", diff.CommitlintValid, diff.PommitlintValid)
	}
}

func TestCompareFindingMismatch(t *testing.T) {
	t.Parallel()

	cl := LintResult{
		ID:    "test",
		Valid: false,
		Findings: []FindingSummary{
			{Rule: "type-enum", Level: "error"},
			{Rule: "type-case", Level: "error"},
		},
	}
	pl := LintResult{
		ID:    "test",
		Valid: false,
		Findings: []FindingSummary{
			{Rule: "type-enum", Level: "error"},
		},
	}

	diff := Compare(cl, pl, "FIX: msg")
	if diff == nil {
		t.Fatal("expected diff, got nil")
	}

	wantOnlyCL := []FindingSummary{{Rule: "type-case", Level: "error"}}
	if d := cmp.Diff(wantOnlyCL, diff.OnlyCommitlint); d != "" {
		t.Errorf("OnlyCommitlint mismatch (-want +got):\n%s", d)
	}

	if len(diff.OnlyPommitlint) != 0 {
		t.Errorf("expected empty OnlyPommitlint, got %+v", diff.OnlyPommitlint)
	}
}

func TestCompareIgnoredMismatch(t *testing.T) {
	t.Parallel()

	cl := LintResult{ID: "test", Valid: true, Ignored: true}
	pl := LintResult{ID: "test", Valid: true, Ignored: false}

	diff := Compare(cl, pl, "Merge branch 'x'")
	if diff == nil {
		t.Fatal("expected diff, got nil")
	}
}

func TestCompareBothEmpty(t *testing.T) {
	t.Parallel()

	cl := LintResult{ID: "test", Valid: true}
	pl := LintResult{ID: "test", Valid: true}

	if diff := Compare(cl, pl, "feat: ok"); diff != nil {
		t.Errorf("expected nil diff, got %+v", diff)
	}
}
