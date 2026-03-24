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
