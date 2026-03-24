package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuymn/pommitlint/internal/preset"
)

func TestNormalizePreset(t *testing.T) {
	t.Parallel()

	raw := rawPreset{
		Rules:        validRawRules(),
		ParserPreset: validRawParserPreset(),
	}

	got, err := normalizePreset(raw)
	if err != nil {
		t.Fatalf("normalizePreset() error = %v", err)
	}

	if got.Version != preset.SchemaVersion {
		t.Fatalf("Version = %d, want %d", got.Version, preset.SchemaVersion)
	}

	if got.Rules[preset.RuleHeaderMaxLength].Applicable != preset.ApplicableAlways {
		t.Fatalf("Applicable = %q", got.Rules[preset.RuleHeaderMaxLength].Applicable)
	}

	var typeEnum []string
	if err := json.Unmarshal(got.Rules[preset.RuleTypeEnum].Value, &typeEnum); err != nil {
		t.Fatalf("unmarshal type-enum: %v", err)
	}

	if len(typeEnum) != 2 || typeEnum[0] != "feat" || typeEnum[1] != "fix" {
		t.Fatalf("type-enum = %#v", typeEnum)
	}
}

func TestNormalizePresetRejectsUnknownRule(t *testing.T) {
	t.Parallel()

	raw := rawPreset{
		Rules: map[string]rawValue{
			"unknown-rule": rawArray(rawNumber(2), rawString("always")),
		},
		ParserPreset: validRawParserPreset(),
	}

	if _, err := normalizePreset(raw); err == nil {
		t.Fatal("normalizePreset() error = nil, want unknown rule error")
	}
}

func TestNormalizePresetRejectsUnsupportedParserField(t *testing.T) {
	t.Parallel()

	parser := validRawParserPreset()
	parser.ParserOpts.Object["mergePattern"] = rawRegexp("^merge$", "")

	raw := rawPreset{
		Rules:        validRawRules(),
		ParserPreset: parser,
	}

	if _, err := normalizePreset(raw); err == nil {
		t.Fatal("normalizePreset() error = nil, want unsupported parser field error")
	}
}

func TestNormalizePresetRejectsFunctionValue(t *testing.T) {
	t.Parallel()

	raw := rawPreset{
		Rules: map[string]rawValue{
			"header-max-length": rawArray(rawNumber(2), rawString("always"), rawFunction()),
		},
		ParserPreset: validRawParserPreset(),
	}

	if _, err := normalizePreset(raw); err == nil {
		t.Fatal("normalizePreset() error = nil, want function error")
	}
}

func TestSyncPresetDoesNotWritePartialArtifact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "preset.json")
	original := []byte("{\"before\":true}\n")
	if err := os.WriteFile(artifactPath, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolver := func(context.Context) (rawPreset, error) {
		return rawPreset{
			Rules: map[string]rawValue{
				"unknown-rule": rawArray(rawNumber(2), rawString("always")),
			},
			ParserPreset: validRawParserPreset(),
		}, nil
	}

	if err := syncPreset(t.Context(), syncOptions{
		ArtifactPath: artifactPath,
		Resolve:      resolver,
	}); err == nil {
		t.Fatal("syncPreset() error = nil, want failure")
	}

	got, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != string(original) {
		t.Fatalf("artifact changed = %q, want %q", string(got), string(original))
	}
}

func validRawRules() map[string]rawValue {
	return map[string]rawValue{
		"body-leading-blank":     rawArray(rawNumber(1), rawString("always")),
		"body-max-line-length":   rawArray(rawNumber(2), rawString("always"), rawNumber(100)),
		"footer-leading-blank":   rawArray(rawNumber(1), rawString("always")),
		"footer-max-line-length": rawArray(rawNumber(2), rawString("always"), rawNumber(100)),
		"header-max-length":      rawArray(rawNumber(2), rawString("always"), rawNumber(100)),
		"header-trim":            rawArray(rawNumber(2), rawString("always")),
		"subject-case": rawArray(
			rawNumber(2),
			rawString("never"),
			rawArray(
				rawString("sentence-case"),
				rawString("start-case"),
				rawString("pascal-case"),
				rawString("upper-case"),
			),
		),
		"subject-empty":     rawArray(rawNumber(2), rawString("never")),
		"subject-full-stop": rawArray(rawNumber(2), rawString("never"), rawString(".")),
		"type-case":         rawArray(rawNumber(2), rawString("always"), rawString("lower-case")),
		"type-empty":        rawArray(rawNumber(2), rawString("never")),
		"type-enum":         rawArray(rawNumber(2), rawString("always"), rawArray(rawString("feat"), rawString("fix"))),
	}
}

func validRawParserPreset() rawParserPreset {
	return rawParserPreset{
		Name: "conventional-changelog-conventionalcommits",
		ParserOpts: rawObject(map[string]rawValue{
			"headerPattern":         rawRegexp("^(\\w*)(?:\\((.*)\\))?!?: (.*)$", ""),
			"breakingHeaderPattern": rawRegexp("^(\\w*)(?:\\((.*)\\))?!: (.*)$", ""),
			"headerCorrespondence":  rawArray(rawString("type"), rawString("scope"), rawString("subject")),
			"noteKeywords":          rawArray(rawString("BREAKING CHANGE"), rawString("BREAKING-CHANGE")),
			"revertPattern": rawRegexp(
				"^(?:Revert|revert:)\\s\"?([\\s\\S]+?)\"?\\s*This reverts commit (\\w*)\\.",
				"i",
			),
			"revertCorrespondence": rawArray(rawString("header"), rawString("hash")),
			"issuePrefixes":        rawArray(rawString("#")),
		}),
	}
}
