package lint_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

func TestCompatConfigConventionalIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		origin   string
		message  string
		valid    bool
		findings []string
	}{
		{
			name:     "type-enum",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "foo: some message",
			valid:    false,
			findings: []string{"error:type-enum"},
		},
		{
			name:     "type-case",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "FIX: some message",
			valid:    false,
			findings: []string{"error:type-case", "error:type-enum"},
		},
		{
			name:     "type-empty",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  ": some message",
			valid:    false,
			findings: []string{"error:type-empty"},
		},
		{
			name:     "subject-case/sentence",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix(scope): Some message",
			valid:    false,
			findings: []string{"error:subject-case"},
		},
		{
			name:     "subject-case/start",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix(scope): Some Message",
			valid:    false,
			findings: []string{"error:subject-case"},
		},
		{
			name:     "subject-case/pascal",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix(scope): SomeMessage",
			valid:    false,
			findings: []string{"error:subject-case"},
		},
		{
			name:     "subject-case/upper",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix(scope): SOMEMESSAGE",
			valid:    false,
			findings: []string{"error:subject-case"},
		},
		{
			name:     "subject-empty",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix:",
			valid:    false,
			findings: []string{"error:subject-empty", "error:type-empty"},
		},
		{
			name:     "subject-full-stop",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix: some message.",
			valid:    false,
			findings: []string{"error:subject-full-stop"},
		},
		{
			name:     "header-max-length",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix: some message that is way too long and breaks the line max-length by several characters since the max is 100",
			valid:    false,
			findings: []string{"error:header-max-length"},
		},
		{
			name:     "footer-leading-blank",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix: some message\n\nbody\nBREAKING CHANGE: It will be significant",
			valid:    true,
			findings: []string{"warning:footer-leading-blank"},
		},
		{
			name:     "footer-max-line-length",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix: some message\n\nbody\n\nBREAKING CHANGE: footer with multiple lines\nhas a message that is way too long and will break the line rule \"line-max-length\" by several characters",
			valid:    false,
			findings: []string{"error:footer-max-line-length"},
		},
		{
			name:     "body-leading-blank",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix: some message\nbody",
			valid:    true,
			findings: []string{"warning:body-leading-blank"},
		},
		{
			name:     "body-max-line-length",
			origin:   "@commitlint/config-conventional/src/index.test.ts",
			message:  "fix: some message\n\nbody with multiple lines\nhas a message that is way too long and will break the line rule \"line-max-length\" by several characters",
			valid:    false,
			findings: []string{"error:body-max-line-length"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema := loadCompatSchema(t)
			result, err := lint.Lint(tt.message, "compat", &schema)
			if err != nil {
				t.Fatalf("%s: Lint() error = %v", tt.origin, err)
			}

			if diff := cmp.Diff(tt.valid, result.Valid); diff != "" {
				t.Fatalf("%s: Valid mismatch (-want +got):\n%s", tt.origin, diff)
			}

			got := summarizeFindings(result.Findings)
			if diff := cmp.Diff(tt.findings, got); diff != "" {
				t.Fatalf("%s: findings mismatch (-want +got):\n%s", tt.origin, diff)
			}
		})
	}
}

func TestCompatConfigConventionalValidMessages(t *testing.T) {
	t.Parallel()

	messages := []string{
		"fix: some message",
		"fix(scope): some message",
		"fix(scope): some Message",
		"fix(scope): some message\n\nBREAKING CHANGE: it will be significant!",
		"fix(scope): some message\n\nbody",
		"fix(scope)!: some message\n\nbody",
	}

	schema := loadCompatSchema(t)
	for _, message := range messages {
		t.Run(strings.ReplaceAll(message, "\n", `\n`), func(t *testing.T) {
			t.Parallel()

			result, err := lint.Lint(message, "compat", &schema)
			if err != nil {
				t.Fatalf("Lint() error = %v", err)
			}

			if !result.Valid {
				t.Fatalf("Valid = false, want true, findings = %#v", result.Findings)
			}
			if len(result.Findings) != 0 {
				t.Fatalf("Findings = %#v, want empty", result.Findings)
			}
		})
	}
}

func TestCompatRuleBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		origin      string
		message     string
		ruleName    preset.RuleName
		rule        preset.Rule
		wantFinding bool
	}{
		{
			name:        "body-leading-blank/no-body",
			origin:      "@commitlint/rules/src/body-leading-blank.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleBodyLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "body-leading-blank/missing-blank",
			origin:      "@commitlint/rules/src/body-leading-blank.test.ts",
			message:     "feat: subject\nbody",
			ruleName:    preset.RuleBodyLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "body-leading-blank/present",
			origin:      "@commitlint/rules/src/body-leading-blank.test.ts",
			message:     "feat: subject\n\nbody",
			ruleName:    preset.RuleBodyLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "body-max-line-length/short",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: subject\n\nshort",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: false,
		},
		{
			name:        "body-max-line-length/long",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: subject\n\nthis line is too long",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: true,
		},
		{
			name:        "body-max-line-length/url",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: subject\n\nhttps://example.com/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/missing-blank",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\nBREAKING CHANGE: body",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "footer-leading-blank/present",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\n\nBREAKING CHANGE: body",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/body-and-footer",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\n\nbody\n\nBREAKING CHANGE: body",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-max-line-length/short",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: subject\n\nBREAKING CHANGE: short",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantFinding: false,
		},
		{
			name:        "footer-max-line-length/long",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: subject\n\nBREAKING CHANGE: this footer line is too long",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantFinding: true,
		},
		{
			name:        "footer-max-line-length/url-not-ignored",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: subject\n\nRefs: https://example.com/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantFinding: true,
		},
		{
			name:        "header-max-length/short",
			origin:      "@commitlint/rules/src/header-max-length.test.ts",
			message:     "feat: a",
			ruleName:    preset.RuleHeaderMaxLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 7)},
			wantFinding: false,
		},
		{
			name:        "header-max-length/exact-boundary",
			origin:      "@commitlint/rules/src/header-max-length.test.ts",
			message:     "feat: ab",
			ruleName:    preset.RuleHeaderMaxLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 8)},
			wantFinding: false,
		},
		{
			name:        "header-max-length/long",
			origin:      "@commitlint/rules/src/header-max-length.test.ts",
			message:     "feat: ab",
			ruleName:    preset.RuleHeaderMaxLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 7)},
			wantFinding: true,
		},
		{
			name:        "header-max-length/emoji-at-boundary",
			origin:      "pommitlint: JS string.length compat (UTF-16 surrogate pair)",
			message:     "feat: 👍",
			ruleName:    preset.RuleHeaderMaxLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 8)},
			wantFinding: false,
		},
		{
			name:        "header-max-length/emoji-exceeds",
			origin:      "pommitlint: JS string.length compat (UTF-16 surrogate pair)",
			message:     "feat: 👍",
			ruleName:    preset.RuleHeaderMaxLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 7)},
			wantFinding: true,
		},
		{
			name:        "header-trim/clean",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "header-trim/leading-whitespace",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     " feat: subject",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "header-trim/trailing-tab",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "feat: subject\t",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "subject-case/lowercase-ascii",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: lowercase",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/sentence-case-ascii",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Sentence case",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/uppercase-cyrillic",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: ПРИВЕТ",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/lowercase-cyrillic",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: привет",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-empty/missing-never",
			origin:      "@commitlint/rules/src/subject-empty.test.ts",
			message:     "feat:",
			ruleName:    preset.RuleSubjectEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},
		{
			name:        "subject-empty/present-never",
			origin:      "@commitlint/rules/src/subject-empty.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantFinding: false,
		},
		{
			name:        "subject-empty/present-always",
			origin:      "@commitlint/rules/src/subject-empty.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "subject-empty/missing-always",
			origin:      "@commitlint/rules/src/subject-empty.test.ts",
			message:     "feat:",
			ruleName:    preset.RuleSubjectEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "subject-full-stop/period",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject.",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantFinding: true,
		},
		{
			name:        "subject-full-stop/no-period",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantFinding: false,
		},
		{
			name:        "subject-full-stop/ellipsis",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject…",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantFinding: false,
		},
		{
			name:        "type-case/lowercase",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: false,
		},
		{
			name:        "type-case/uppercase",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FEAT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/camelcase-vs-uppercase",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "featScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "upper-case")},
			wantFinding: true,
		},
		{
			name:        "type-empty/missing-never",
			origin:      "@commitlint/rules/src/type-empty.test.ts",
			message:     ": subject",
			ruleName:    preset.RuleTypeEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},
		{
			name:        "type-empty/present-never",
			origin:      "@commitlint/rules/src/type-empty.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleTypeEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantFinding: false,
		},
		{
			name:        "type-empty/present-always",
			origin:      "@commitlint/rules/src/type-empty.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleTypeEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "type-empty/scope-only-never",
			origin:      "@commitlint/rules/src/type-empty.test.ts",
			message:     "(scope): subject",
			ruleName:    preset.RuleTypeEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},
		{
			name:        "type-enum/allowed",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: false,
		},
		{
			name:        "type-enum/disallowed",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "docs: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: true,
		},
		{
			name:        "type-enum/empty",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     ": subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: false,
		},

		// subject-case: config-conventional preset (never [sentence-case, start-case, pascal-case, upper-case])
		{
			name:        "subject-case/mixedcase-not-forbidden",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/camelcase-not-forbidden",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subJect",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/kebabcase-not-forbidden",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sub-ject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/snakecase-not-forbidden",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sub_ject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/startcase-forbidden",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Sub Ject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/sentencecase-forbidden",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Sub ject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: true,
		},

		// subject-case: array values with always/never
		{
			name:        "subject-case/always-multi-pass-upper",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: SUBJECT",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-multi-pass-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-multi-fail-mixed",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/never-multi-pass-mixed",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"upper-case", "lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/never-multi-fail-upper",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: SUBJECT",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"upper-case", "lower-case"})},
			wantFinding: true,
		},

		// subject-case: unicode variants
		{
			name:        "subject-case/uppercase-frisian",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: ÛNDERWERP",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/lowercase-tajik",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: мав зуъ",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},

		// header-trim: whitespace variations
		{
			name:        "header-trim/trailing-spaces",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "feat: subject  ",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "header-trim/both-ends-space",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     " feat: subject ",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "header-trim/leading-tabs",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "\t\tfeat: subject",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "header-trim/both-ends-tab",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "\t\tfeat: subject\t",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},
		{
			name:        "header-trim/mixed-whitespace-surround",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "\t feat: subject \t",
			ruleName:    preset.RuleHeaderTrim,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways},
			wantFinding: true,
		},

		// subject-full-stop: edge cases
		{
			name:        "subject-full-stop/empty-subject",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: ",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantFinding: false,
		},
		{
			name:        "subject-full-stop/scoped-with-period",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "fix(scope): subject.",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantFinding: true,
		},
		{
			name:        "subject-full-stop/always-with-period",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject.",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, ".")},
			wantFinding: false,
		},
		{
			name:        "subject-full-stop/always-without-period",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, ".")},
			wantFinding: true,
		},
		{
			name:        "subject-full-stop/ascii-ellipsis-not-fullstop",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject ends with ellipsis...",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantFinding: false,
		},

		// footer-leading-blank: expanded coverage
		{
			name:        "footer-leading-blank/no-body-footer-only",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\n\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/multiline-body",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\n\nbody1\nbody2\n\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/never-with-blank",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\n\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},
		{
			name:        "footer-leading-blank/never-without-blank",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: subject\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: false,
		},

		// body-max-line-length: multi-line and URL in context
		{
			name:        "body-max-line-length/multi-short",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub\n\na\na\na",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: false,
		},
		{
			name:        "body-max-line-length/multi-one-long",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub\n\na\nthis is long\na",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: true,
		},
		{
			name:        "body-max-line-length/markdown-url",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub\n\nSee [link](https://example.com/very/long/path/exceeding).",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: false,
		},
		{
			name:        "body-max-line-length/no-body",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantFinding: false,
		},
		{
			name:        "body-max-line-length/limit-1-pass",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub\n\na",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 1)},
			wantFinding: false,
		},
		{
			name:        "body-max-line-length/limit-1-fail",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub\n\nab",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 1)},
			wantFinding: true,
		},

		// body-leading-blank: never applicable
		{
			name:        "body-leading-blank/never-blank-present",
			origin:      "@commitlint/rules/src/body-leading-blank.test.ts",
			message:     "feat: subject\n\nbody",
			ruleName:    preset.RuleBodyLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},
		{
			name:        "body-leading-blank/never-blank-missing",
			origin:      "@commitlint/rules/src/body-leading-blank.test.ts",
			message:     "feat: subject\nbody",
			ruleName:    preset.RuleBodyLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: false,
		},

		// type-case: additional patterns
		{
			name:        "type-case/empty-type",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     ": subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: false,
		},
		{
			name:        "type-case/pascalcase-type",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "TyPe: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: true,
		},

		// type-enum: never applicable and empty type
		{
			name:        "type-enum/never-in-list",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: true,
		},
		{
			name:        "type-enum/never-not-in-list",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "docs: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: false,
		},
		{
			name:        "type-enum/always-empty-type",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     ": subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: false,
		},

		// footer-leading-blank: no-footer and trailing newline
		{
			name:        "footer-leading-blank/body-only-no-footer-always",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody paragraph",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/body-only-no-footer-never",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody paragraph",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/trailing-newline-always",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody\n",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/trailing-newline-never",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody\n",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/body-and-footer-never",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody\n\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},
		{
			name:        "footer-leading-blank/double-blank-always",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody\n\n\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantFinding: false,
		},
		{
			name:        "footer-leading-blank/double-blank-never",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\n\nbody\n\n\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableNever},
			wantFinding: true,
		},

		// footer-max-line-length: multi-line footer
		{
			name:        "footer-max-line-length/multi-all-short",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: sub\n\nBREAKING CHANGE: short\nmore text",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantFinding: false,
		},
		{
			name:        "footer-max-line-length/multi-one-long",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: sub\n\nBREAKING CHANGE: ok\nthis second footer line is way too long for the limit",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantFinding: true,
		},
		{
			name:        "footer-max-line-length/no-footer",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantFinding: false,
		},

		// type-enum: expanded coverage
		{
			name:        "type-enum/never-empty-type",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     ": subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"feat", "fix"})},
			wantFinding: false,
		},
		{
			name:        "type-enum/always-multiple-values",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix", "docs", "chore"})},
			wantFinding: false,
		},
		{
			name:        "type-enum/always-not-in-large-list",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "perf: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix", "docs", "chore"})},
			wantFinding: true,
		},

		// type-case: camelcase type variations
		{
			name:        "type-case/camelcase-always-lower",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "featScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/camelcase-always-upper",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "featScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "upper-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/camelcase-always-sentence",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "featScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "sentence-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/camelcase-always-pascal",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "featScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "pascal-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/camelcase-always-camelcase",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "featScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "camel-case")},
			wantFinding: false,
		},

		// type-case: pascalcase type variations
		{
			name:        "type-case/pascalcase-always-lower",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FeatScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/pascalcase-always-upper",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FeatScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "upper-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/pascalcase-always-pascal",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FeatScope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "pascal-case")},
			wantFinding: false,
		},

		// type-case: snakecase type variations
		{
			name:        "type-case/snakecase-always-lower",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "feat_scope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: false,
		},
		{
			name:        "type-case/snakecase-always-upper",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "feat_scope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "upper-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/snakecase-always-snake",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "feat_scope: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "snake-case")},
			wantFinding: false,
		},

		// type-case: never applicable and mixed case
		{
			name:        "type-case/uppercase-never-lower",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FEAT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, "lower-case")},
			wantFinding: false,
		},
		{
			name:        "type-case/uppercase-never-upper",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FEAT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, "upper-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/mixedcase-always-lower",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "fEaT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/mixedcase-always-upper",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "fEaT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "upper-case")},
			wantFinding: true,
		},
		{
			name:        "type-case/mixedcase-never-lower",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "fEaT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, "lower-case")},
			wantFinding: false,
		},
		{
			name:        "type-case/mixedcase-never-upper",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "fEaT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, "upper-case")},
			wantFinding: false,
		},

		// subject-case: array values expanded
		{
			name:        "subject-case/always-triple-pass-sentence",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Some subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case", "sentence-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-triple-fail-mixed",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sOmE sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case", "sentence-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/never-triple-pass-mixed",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sOmE sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"upper-case", "lower-case", "sentence-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/never-triple-fail-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: some subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"upper-case", "lower-case", "sentence-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/always-unsupported-only",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subJect",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"camel-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/never-unsupported-only",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subJect",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"camel-case"})},
			wantFinding: false,
		},

		// subject-case: individual case always positive/negative paths
		{
			name:        "subject-case/always-lower-pass",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-upper-pass",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: SUBJECT",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-pascal-pass",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: SubJect",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"pascal-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-start-pass",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Sub Ject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"start-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-sentence-pass",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Sub ject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"sentence-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/always-lower-fail",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/always-upper-fail",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/always-sentence-fail",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: subject",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"sentence-case"})},
			wantFinding: true,
		},

		// subject-case: Unicode scripts with cased letters
		{
			name:        "subject-case/bulgarian-lower-always-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: тема",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/bulgarian-lower-never-upper",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: тема",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/kazakh-pascal-always-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: ТақыРып",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/kazakh-pascal-always-upper",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: ТақыРып",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/swedish-start-always-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Äm Ne",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/mixed-latin-cyrillic-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "chore: update зависимости",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/cyrillic-upper-never-multi",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "chore: ОБНОВЛЕНЫ ВСЕ ЗАВИСИМОСТИ",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/chinese-always-upper-caseless",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: 这是一次提交",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/irish-kebab-always-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: áb-har",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/mongolian-snake-always-lower",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: сэ_дэв",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantFinding: false,
		},

		// subject-case: Unicode case-format matching
		{
			name:        "subject-case/tajik-sentence-always-pass",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: Мав зуъ",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"sentence-case"})},
			wantFinding: false,
		},
		{
			name:        "subject-case/swedish-start-always-ascii-only",
			origin:      "pommitlint: isStartCase uses ASCII-only word boundaries",
			message:     "feat: Äm Ne",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"start-case"})},
			wantFinding: true,
		},
		{
			name:        "subject-case/greek-camel-unsupported",
			origin:      "pommitlint: camel-case not in subjectCaseMatchers",
			message:     "feat: θέΜα",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"camel-case"})},
			wantFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rules := make(map[preset.RuleName]preset.Rule, 1)
			rules[tt.ruleName] = tt.rule
			schema := compatSchemaForRules(t, rules)
			result, err := lint.Lint(tt.message, "compat", &schema)
			if err != nil {
				t.Fatalf("%s: Lint() error = %v", tt.origin, err)
			}

			gotFinding := hasRule(result.Findings, tt.ruleName)
			if diff := cmp.Diff(tt.wantFinding, gotFinding); diff != "" {
				t.Fatalf("%s: finding presence mismatch (-want +got):\n%s\nfindings=%#v", tt.origin, diff, result.Findings)
			}
		})
	}
}

func TestCompatParseScopeAndFooterBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		origin          string
		message         string
		wantScope       string
		wantBodyLines   []string
		wantFooterLines []string
	}{
		{
			name:            "scope-with-slash",
			origin:          "@commitlint/parse/src/index.test.ts",
			message:         "type(some/scope): subject",
			wantScope:       "some/scope",
			wantBodyLines:   nil,
			wantFooterLines: nil,
		},
		{
			name:            "scope-with-comma",
			origin:          "@commitlint/parse/src/index.test.ts",
			message:         "type(component,demo): subject",
			wantScope:       "component,demo",
			wantBodyLines:   nil,
			wantFooterLines: nil,
		},
		{
			name:            "scope-with-chinese-characters",
			origin:          "@commitlint/parse/src/index.test.ts",
			message:         "fix(面试评价): 测试",
			wantScope:       "面试评价",
			wantBodyLines:   nil,
			wantFooterLines: nil,
		},
		{
			name:            "inline-reference-moves-body-content-to-footer",
			origin:          "@commitlint/parse/src/index.test.ts",
			message:         "type(some/scope): subject #reference\n\nthings #reference",
			wantScope:       "some/scope",
			wantBodyLines:   []string{},
			wantFooterLines: []string{"things #reference"},
		},
		{
			name:            "double-newline-body-and-closes",
			origin:          "@commitlint/parse/src/index.test.ts",
			message:         "fix: issue\n\ndetailed explanation\n\nCloses: #123",
			wantScope:       "",
			wantBodyLines:   []string{"detailed explanation"},
			wantFooterLines: []string{"Closes: #123"},
		},
	}

	schema := loadCompatSchema(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			message, err := lint.Parse(tt.message, &schema)
			if err != nil {
				t.Fatalf("%s: Parse() error = %v", tt.origin, err)
			}

			if diff := cmp.Diff(tt.wantScope, message.Scope); diff != "" {
				t.Fatalf("%s: Scope mismatch (-want +got):\n%s", tt.origin, diff)
			}
			if diff := cmp.Diff(tt.wantBodyLines, message.BodyLines); diff != "" {
				t.Fatalf("%s: BodyLines mismatch (-want +got):\n%s", tt.origin, diff)
			}
			if diff := cmp.Diff(tt.wantFooterLines, message.FooterLines); diff != "" {
				t.Fatalf("%s: FooterLines mismatch (-want +got):\n%s", tt.origin, diff)
			}
		})
	}
}

func TestCompatSubjectCaseSkipsNonCasedSubjects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origin  string
		message string
	}{
		{
			name:    "empty-subject",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "feat:",
		},
		{
			name:    "numeric-subject",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "feat: 1.0.0",
		},
		{
			name:    "caseless-subject",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "feat: 这是一次提交",
		},
		{
			name:    "non-latin-subject",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "feat: 追加する",
		},
		{
			name:    "arabic-rtl-subject",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "feat: إضافة وظيفة جديدة",
		},
		{
			name:    "hebrew-rtl-subject",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "fix: תיקון בעיה",
		},
		{
			name:    "arabic-rtl-with-scope",
			origin:  "@commitlint/rules/src/subject-case.test.ts",
			message: "feat(مميزات): إضافة وظيفة جديدة",
		},
	}

	rules := make(map[preset.RuleName]preset.Rule, 1)
	rules[preset.RuleSubjectCase] = preset.Rule{
		Level:      2,
		Applicable: preset.ApplicableNever,
		Value: rawJSON(t, []string{
			"sentence-case",
			"start-case",
			"pascal-case",
			"upper-case",
		}),
	}
	schema := compatSchemaForRules(t, rules)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := lint.Lint(tt.message, "compat", &schema)
			if err != nil {
				t.Fatalf("%s: Lint() error = %v", tt.origin, err)
			}

			if hasRule(result.Findings, preset.RuleSubjectCase) {
				t.Fatalf("%s: unexpected subject-case finding: %#v", tt.origin, result.Findings)
			}
		})
	}
}

func TestCompatHeaderTrimMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		origin      string
		message     string
		wantMessage string
	}{
		{
			name:        "leading-whitespace",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     " feat: subject",
			wantMessage: "header must not start with whitespace",
		},
		{
			name:        "trailing-whitespace",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "feat: subject  ",
			wantMessage: "header must not end with whitespace",
		},
		{
			name:        "surrounded-whitespace",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     " feat: subject ",
			wantMessage: "header must not be surrounded by whitespace",
		},
		{
			name:        "leading-tab",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "\tfeat: subject",
			wantMessage: "header must not start with whitespace",
		},
		{
			name:        "trailing-tab",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "feat: subject\t",
			wantMessage: "header must not end with whitespace",
		},
		{
			name:        "surrounded-tab",
			origin:      "@commitlint/rules/src/header-trim.test.ts",
			message:     "\tfeat: subject\t",
			wantMessage: "header must not be surrounded by whitespace",
		},
	}

	rules := make(map[preset.RuleName]preset.Rule, 1)
	rules[preset.RuleHeaderTrim] = preset.Rule{Level: 2, Applicable: preset.ApplicableAlways}
	schema := compatSchemaForRules(t, rules)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := lint.Lint(tt.message, "compat", &schema)
			if err != nil {
				t.Fatalf("%s: Lint() error = %v", tt.origin, err)
			}

			gotMessage := findingMessage(result.Findings, preset.RuleHeaderTrim)
			if gotMessage == "" {
				t.Fatalf("%s: no header-trim finding", tt.origin)
			}

			if diff := cmp.Diff(tt.wantMessage, gotMessage); diff != "" {
				t.Fatalf("%s: message mismatch (-want +got):\n%s", tt.origin, diff)
			}
		})
	}
}

func TestCompatEmptyMessageLint(t *testing.T) {
	t.Parallel()

	// origin: @commitlint/lint/src/lint.test.ts ("positive on empty message")
	rules := make(map[preset.RuleName]preset.Rule)
	schema := compatSchemaForRules(t, rules)
	result, err := lint.Lint("", "compat", &schema)
	if err != nil {
		t.Fatalf("Lint() error = %v", err)
	}

	if !result.Valid {
		t.Fatalf("Valid = false, want true, findings = %#v", result.Findings)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("Findings = %#v, want empty", result.Findings)
	}
}

func TestCompatFindingMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		origin      string
		message     string
		ruleName    preset.RuleName
		rule        preset.Rule
		wantMessage string
	}{
		{
			name:        "body-leading-blank",
			origin:      "@commitlint/rules/src/body-leading-blank.test.ts",
			message:     "feat: sub\nbody",
			ruleName:    preset.RuleBodyLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantMessage: "body must begin with a blank line",
		},
		{
			name:        "body-max-line-length",
			origin:      "@commitlint/rules/src/body-max-line-length.test.ts",
			message:     "feat: sub\n\nthis line is too long",
			ruleName:    preset.RuleBodyMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 10)},
			wantMessage: "body line 1 exceeds max length 10",
		},
		{
			name:        "footer-leading-blank",
			origin:      "@commitlint/rules/src/footer-leading-blank.test.ts",
			message:     "feat: sub\nBREAKING CHANGE: desc",
			ruleName:    preset.RuleFooterLeadingBlank,
			rule:        preset.Rule{Level: 1, Applicable: preset.ApplicableAlways},
			wantMessage: "footer must begin with a blank line",
		},
		{
			name:        "footer-max-line-length",
			origin:      "@commitlint/rules/src/footer-max-line-length.test.ts",
			message:     "feat: sub\n\nBREAKING CHANGE: this is way too long",
			ruleName:    preset.RuleFooterMaxLineLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 25)},
			wantMessage: "footer line 1 exceeds max length 25",
		},
		{
			name:        "header-max-length",
			origin:      "@commitlint/rules/src/header-max-length.test.ts",
			message:     "feat: ab",
			ruleName:    preset.RuleHeaderMaxLength,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, 7)},
			wantMessage: "header exceeds max length 7",
		},
		{
			name:        "subject-empty",
			origin:      "@commitlint/rules/src/subject-empty.test.ts",
			message:     "feat:",
			ruleName:    preset.RuleSubjectEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantMessage: "subject may not be empty",
		},
		{
			name:        "subject-full-stop",
			origin:      "@commitlint/rules/src/subject-full-stop.test.ts",
			message:     "feat: subject.",
			ruleName:    preset.RuleSubjectFullStop,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, ".")},
			wantMessage: `subject may not end with "."`,
		},
		{
			name:        "type-empty",
			origin:      "@commitlint/rules/src/type-empty.test.ts",
			message:     ": subject",
			ruleName:    preset.RuleTypeEmpty,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever},
			wantMessage: "type may not be empty",
		},
		{
			name:        "type-case",
			origin:      "@commitlint/rules/src/type-case.test.ts",
			message:     "FEAT: subject",
			ruleName:    preset.RuleTypeCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, "lower-case")},
			wantMessage: "type must be lower-case",
		},
		{
			name:        "type-enum",
			origin:      "@commitlint/rules/src/type-enum.test.ts",
			message:     "docs: subject",
			ruleName:    preset.RuleTypeEnum,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"feat", "fix"})},
			wantMessage: "type must be one of: feat, fix",
		},

		// subject-case: joinCases formatting verification
		{
			name:        "subject-case/msg-single",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: SUBJECT",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"lower-case"})},
			wantMessage: "subject must be lower-case",
		},
		{
			name:        "subject-case/msg-two",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case"})},
			wantMessage: "subject must be upper-case or lower-case",
		},
		{
			name:        "subject-case/msg-three",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: sUbJeCt",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableAlways, Value: rawJSON(t, []string{"upper-case", "lower-case", "sentence-case"})},
			wantMessage: "subject must be upper-case, lower-case, or sentence-case",
		},
		{
			name:        "subject-case/msg-never-four",
			origin:      "@commitlint/rules/src/subject-case.test.ts",
			message:     "feat: SUBJECT",
			ruleName:    preset.RuleSubjectCase,
			rule:        preset.Rule{Level: 2, Applicable: preset.ApplicableNever, Value: rawJSON(t, []string{"sentence-case", "start-case", "pascal-case", "upper-case"})},
			wantMessage: "subject must not be sentence-case, start-case, pascal-case, or upper-case",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rules := make(map[preset.RuleName]preset.Rule, 1)
			rules[tt.ruleName] = tt.rule
			schema := compatSchemaForRules(t, rules)
			result, err := lint.Lint(tt.message, "compat", &schema)
			if err != nil {
				t.Fatalf("%s: Lint() error = %v", tt.origin, err)
			}

			gotMessage := findingMessage(result.Findings, tt.ruleName)
			if gotMessage == "" {
				t.Fatalf("%s: no %s finding, findings = %#v", tt.origin, tt.ruleName, result.Findings)
			}

			if diff := cmp.Diff(tt.wantMessage, gotMessage); diff != "" {
				t.Fatalf("%s: message mismatch (-want +got):\n%s", tt.origin, diff)
			}
		})
	}
}

func loadCompatSchema(t *testing.T) preset.Schema {
	t.Helper()

	schema, err := preset.Load()
	if err != nil {
		t.Fatalf("preset.Load() error = %v", err)
	}

	return schema
}

func compatSchemaForRules(t *testing.T, rules map[preset.RuleName]preset.Rule) preset.Schema {
	t.Helper()

	base := loadCompatSchema(t)
	return preset.Schema{
		Version:      base.Version,
		Source:       base.Source,
		ParserPreset: base.ParserPreset,
		Rules:        rules,
	}
}

func rawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}

	return encoded
}

func summarizeFindings(findings []lint.Finding) []string {
	result := make([]string, 0, len(findings))
	for _, finding := range findings {
		result = append(result, fmt.Sprintf("%s:%s", finding.Level, finding.Rule))
	}

	return result
}

func hasRule(findings []lint.Finding, ruleName preset.RuleName) bool {
	for _, finding := range findings {
		if finding.Rule == ruleName {
			return true
		}
	}

	return false
}

func findingMessage(findings []lint.Finding, ruleName preset.RuleName) string {
	for _, finding := range findings {
		if finding.Rule == ruleName {
			return finding.Message
		}
	}

	return ""
}
