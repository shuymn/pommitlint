package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"

	"github.com/shuymn/pommitlint/internal/preset"
)

type rawKind string

const (
	rawKindArray    rawKind = "array"
	rawKindBool     rawKind = "bool"
	rawKindFunction rawKind = "function"
	rawKindNull     rawKind = "null"
	rawKindNumber   rawKind = "number"
	rawKindObject   rawKind = "object"
	rawKindRegexp   rawKind = "regexp"
	rawKindString   rawKind = "string"
)

var (
	errFunctionValue = errors.New("non-normalizable function value")
	parserFieldNames = []string{
		"breakingHeaderPattern",
		"headerCorrespondence",
		"headerPattern",
		"issuePrefixes",
		"noteKeywords",
		"revertCorrespondence",
		"revertPattern",
	}
)

type rawPreset struct {
	Rules        map[string]rawValue `json:"rules"`
	ParserPreset rawParserPreset     `json:"parserPreset"` //nolint:ptrstruct // JSON deserialization field: pointer adds nil checks at all access sites with no benefit
}

type rawParserPreset struct {
	Name       string   `json:"name"`
	ParserOpts rawValue `json:"parserOpts"` //nolint:ptrstruct // JSON deserialization field: pointer adds nil checks at all access sites with no benefit
}

type rawValue struct {
	Kind   rawKind             `json:"kind"`
	Bool   bool                `json:"bool,omitempty"`
	Number json.Number         `json:"number,omitempty"`
	String string              `json:"string,omitempty"`
	Items  []rawValue          `json:"items,omitempty"`
	Object map[string]rawValue `json:"object,omitempty"`
	Source string              `json:"source,omitempty"`
	Flags  string              `json:"flags,omitempty"`
}

type syncOptions struct {
	ArtifactPath string
	Resolve      func(context.Context) (rawPreset, error)
}

func syncPreset(ctx context.Context, options syncOptions) error { //nolint:ptrstruct // syncOptions is ~32B; value passing is appropriate
	raw, err := options.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("resolve preset: %w", err)
	}

	normalized, err := normalizePreset(raw)
	if err != nil {
		return fmt.Errorf("normalize preset: %w", err)
	}

	payload, err := marshalSchema(normalized)
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	if err := writeAtomically(options.ArtifactPath, payload); err != nil {
		return fmt.Errorf("write preset artifact: %w", err)
	}

	return nil
}

func resolveWithBun(ctx context.Context) (rawPreset, error) {
	scriptDir := toolDir()
	cmd := exec.CommandContext(ctx, "bun", "run", "resolve-preset.ts")
	cmd.Dir = scriptDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if s := stderr.String(); s != "" {
			return rawPreset{}, fmt.Errorf("bun resolver: %w\nstderr: %s", err, s)
		}
		return rawPreset{}, fmt.Errorf("bun resolver: %w", err)
	}

	var raw rawPreset
	if err := json.Unmarshal(output, &raw); err != nil {
		return rawPreset{}, fmt.Errorf("decode bun output: %w", err)
	}

	return raw, nil
}

func normalizePreset(raw rawPreset) (preset.Schema, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +20% allocs/op
	rules, err := normalizeRules(raw.Rules)
	if err != nil {
		return preset.Schema{}, err
	}

	parserPreset, err := normalizeParserPreset(raw.ParserPreset)
	if err != nil {
		return preset.Schema{}, err
	}

	return preset.Schema{
		Version: preset.SchemaVersion,
		Source: preset.Source{
			ConfigPackage:       "@commitlint/config-conventional",
			ParserPresetPackage: "conventional-changelog-conventionalcommits",
		},
		Rules:        rules,
		ParserPreset: parserPreset,
	}, nil
}

func normalizeRules(rawRules map[string]rawValue) (map[preset.RuleName]preset.Rule, error) {
	knownRules := preset.KnownRules()
	if len(rawRules) != len(knownRules) {
		return nil, fmt.Errorf("rule count mismatch: got %d want %d", len(rawRules), len(knownRules))
	}

	rules := make(map[preset.RuleName]preset.Rule, len(rawRules))
	for name, rawRule := range rawRules {
		ruleName := preset.RuleName(name)
		if _, ok := knownRules[ruleName]; !ok {
			return nil, fmt.Errorf("unknown rule %q", name)
		}

		rule, err := normalizeRule(name, rawRule)
		if err != nil {
			return nil, err
		}

		rules[ruleName] = rule
	}

	for ruleName := range knownRules {
		if _, ok := rules[ruleName]; !ok {
			return nil, fmt.Errorf("missing required rule %q", ruleName)
		}
	}

	return rules, nil
}

func normalizeRule(name string, rawRule rawValue) (preset.Rule, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +20% allocs/op
	if rawRule.Kind != rawKindArray {
		return preset.Rule{}, fmt.Errorf("rule %q must be an array", name)
	}

	if len(rawRule.Items) != 2 && len(rawRule.Items) != 3 {
		return preset.Rule{}, fmt.Errorf("rule %q must have 2 or 3 items", name)
	}

	level, err := rawRule.Items[0].asInt()
	if err != nil {
		return preset.Rule{}, fmt.Errorf("rule %q level: %w", name, err)
	}

	applicable, err := rawRule.Items[1].asApplicable()
	if err != nil {
		return preset.Rule{}, fmt.Errorf("rule %q applicable: %w", name, err)
	}

	rule := preset.Rule{
		Level:      level,
		Applicable: applicable,
	}

	if len(rawRule.Items) == 3 {
		value, err := normalizeRuleValue(rawRule.Items[2])
		if err != nil {
			return preset.Rule{}, fmt.Errorf("rule %q value: %w", name, err)
		}

		rule.Value = value
	}

	return rule, nil
}

func normalizeRuleValue(value rawValue) (json.RawMessage, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +83% allocs/op in normalizeJSONValue
	normalized, err := normalizeJSONValue(value)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode rule value: %w", err)
	}

	return payload, nil
}

func normalizeJSONValue(value rawValue) (any, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +83% allocs/op
	switch value.Kind {
	case rawKindNull:
		return nil, nil
	case rawKindBool:
		return value.Bool, nil
	case rawKindNumber:
		parsed, err := value.Number.Float64()
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", value.Number.String(), err)
		}

		if float64(int(parsed)) == parsed {
			return int(parsed), nil
		}

		return parsed, nil
	case rawKindString:
		return value.String, nil
	case rawKindArray:
		items := make([]any, 0, len(value.Items))
		for _, item := range value.Items {
			normalized, err := normalizeJSONValue(item)
			if err != nil {
				return nil, err
			}

			items = append(items, normalized)
		}

		return items, nil
	case rawKindFunction:
		return nil, errFunctionValue
	case rawKindRegexp:
		return nil, errors.New("regexp values are not supported in rule values")
	case rawKindObject:
		return nil, errors.New("object values are not supported in rule values")
	default:
		return nil, fmt.Errorf("unsupported value kind %q", value.Kind)
	}
}

func normalizeParserPreset(raw rawParserPreset) (preset.ParserPreset, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +20% allocs/op
	if raw.Name == "" {
		return preset.ParserPreset{}, errors.New("parser preset name is required")
	}

	if raw.ParserOpts.Kind != rawKindObject {
		return preset.ParserPreset{}, errors.New("parser preset options must be an object")
	}

	for field := range raw.ParserOpts.Object {
		if !slices.Contains(parserFieldNames, field) {
			return preset.ParserPreset{}, fmt.Errorf("unsupported parser field %q", field)
		}
	}

	for _, field := range parserFieldNames {
		if _, ok := raw.ParserOpts.Object[field]; !ok {
			return preset.ParserPreset{}, fmt.Errorf("missing parser field %q", field)
		}
	}

	headerPattern, err := normalizeRegexp(raw.ParserOpts.Object["headerPattern"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("headerPattern: %w", err)
	}

	breakingHeaderPattern, err := normalizeRegexp(raw.ParserOpts.Object["breakingHeaderPattern"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("breakingHeaderPattern: %w", err)
	}

	headerCorrespondence, err := normalizeStringArray(raw.ParserOpts.Object["headerCorrespondence"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("headerCorrespondence: %w", err)
	}

	noteKeywords, err := normalizeStringArray(raw.ParserOpts.Object["noteKeywords"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("noteKeywords: %w", err)
	}

	revertPattern, err := normalizeRegexp(raw.ParserOpts.Object["revertPattern"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("revertPattern: %w", err)
	}

	revertCorrespondence, err := normalizeStringArray(raw.ParserOpts.Object["revertCorrespondence"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("revertCorrespondence: %w", err)
	}

	issuePrefixes, err := normalizeStringArray(raw.ParserOpts.Object["issuePrefixes"])
	if err != nil {
		return preset.ParserPreset{}, fmt.Errorf("issuePrefixes: %w", err)
	}

	return preset.ParserPreset{
		Name:                  raw.Name,
		HeaderPattern:         headerPattern,
		BreakingHeaderPattern: breakingHeaderPattern,
		HeaderCorrespondence:  headerCorrespondence,
		NoteKeywords:          noteKeywords,
		RevertPattern:         revertPattern,
		RevertCorrespondence:  revertCorrespondence,
		IssuePrefixes:         issuePrefixes,
	}, nil
}

func normalizeRegexp(value rawValue) (preset.Regexp, error) { //nolint:ptrstruct // called with map values (raw.ParserOpts.Object[key]); map values are not addressable in Go
	if value.Kind == rawKindFunction {
		return preset.Regexp{}, errFunctionValue
	}

	if value.Kind != rawKindRegexp {
		return preset.Regexp{}, fmt.Errorf("expected regexp, got %q", value.Kind)
	}

	return preset.Regexp{
		Source: value.Source,
		Flags:  value.Flags,
	}, nil
}

func normalizeStringArray(value rawValue) ([]string, error) { //nolint:ptrstruct // called with map values (raw.ParserOpts.Object[key]); map values are not addressable in Go
	if value.Kind == rawKindFunction {
		return nil, errFunctionValue
	}

	if value.Kind != rawKindArray {
		return nil, fmt.Errorf("expected array, got %q", value.Kind)
	}

	items := make([]string, 0, len(value.Items))
	for _, item := range value.Items {
		if item.Kind == rawKindFunction {
			return nil, errFunctionValue
		}

		if item.Kind != rawKindString {
			return nil, fmt.Errorf("expected string array, got item kind %q", item.Kind)
		}

		items = append(items, item.String)
	}

	return items, nil
}

func marshalSchema(schema preset.Schema) ([]byte, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +20% allocs/op
	payload, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(payload, '\n'), nil
}

func writeAtomically(path string, payload []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	current, err := os.ReadFile(path)
	if err == nil && bytes.Equal(current, payload) {
		return nil
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "preset-*.json")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

func toolDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}

	return filepath.Dir(file)
}

func (value rawValue) asApplicable() (preset.Applicable, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +20% allocs/op
	if value.Kind == rawKindFunction {
		return "", errFunctionValue
	}

	if value.Kind != rawKindString {
		return "", fmt.Errorf("expected string, got %q", value.Kind)
	}

	applicable := preset.Applicable(value.String)
	if applicable != preset.ApplicableAlways && applicable != preset.ApplicableNever {
		return "", fmt.Errorf("unsupported applicable value %q", value.String)
	}

	return applicable, nil
}

func (value rawValue) asInt() (int, error) { //nolint:ptrstruct // pointer forces heap allocation via escape analysis; benchmarked: +20% allocs/op
	if value.Kind == rawKindFunction {
		return 0, errFunctionValue
	}

	if value.Kind != rawKindNumber {
		return 0, fmt.Errorf("expected number, got %q", value.Kind)
	}

	parsed, err := strconv.Atoi(value.Number.String())
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", value.Number.String(), err)
	}

	return parsed, nil
}

func rawArray(items ...rawValue) rawValue {
	return rawValue{
		Kind:  rawKindArray,
		Items: items,
	}
}

func rawFunction() rawValue {
	return rawValue{Kind: rawKindFunction}
}

func rawNumber(value int) rawValue {
	return rawValue{
		Kind:   rawKindNumber,
		Number: json.Number(strconv.Itoa(value)),
	}
}

func rawObject(object map[string]rawValue) rawValue {
	return rawValue{
		Kind:   rawKindObject,
		Object: object,
	}
}

func rawRegexp(source, flags string) rawValue {
	return rawValue{
		Kind:   rawKindRegexp,
		Source: source,
		Flags:  flags,
	}
}

func rawString(value string) rawValue {
	return rawValue{
		Kind:   rawKindString,
		String: value,
	}
}
