package lint

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/shuymn/pommitlint/internal/preset"
)

type Level string

const (
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

const subjectEllipsis = "..."

type Finding struct {
	Rule    preset.RuleName `json:"rule"`
	Level   Level           `json:"level"`
	Field   string          `json:"field"`
	Message string          `json:"message"`
}

type Result struct {
	Source   string    `json:"source"`
	Valid    bool      `json:"valid"`
	Ignored  bool      `json:"ignored"`
	Findings []Finding `json:"findings"`
}

func (r *Result) ErrorCount() int {
	return countByLevel(r.Findings, LevelError)
}

func (r *Result) WarningCount() int {
	return countByLevel(r.Findings, LevelWarning)
}

type Message struct {
	Header             string
	BodyLines          []string
	FooterLines        []string
	BodyLeadingBlank   bool
	FooterLeadingBlank bool
	Type               string
	Scope              string
	Subject            string
}

func Parse(raw string, schema *preset.Schema) (Message, error) {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if normalized == "" {
		return Message{}, nil
	}

	lines := strings.Split(normalized, "\n")
	message := Message{Header: lines[0]}
	if len(lines) == 1 {
		return message, enrichHeader(&message, schema)
	}

	rest := lines[1:]
	headerSeparated := len(rest) > 0 && rest[0] == ""
	message.BodyLeadingBlank = headerSeparated

	footerStart := findFooterStart(rest)
	if footerStart >= 0 {
		message.FooterLeadingBlank = footerStart > 0 && rest[footerStart-1] == ""
		bodyLines := rest[:footerStart]
		if headerSeparated && len(bodyLines) > 0 && bodyLines[0] == "" {
			bodyLines = bodyLines[1:]
		}
		message.BodyLines = trimTrailingBlankLines(bodyLines)
		message.FooterLines = trimTrailingBlankLines(rest[footerStart:])
	} else {
		if headerSeparated {
			rest = rest[1:]
		}
		message.BodyLines = trimTrailingBlankLines(rest)
	}

	return message, enrichHeader(&message, schema)
}

func enrichHeader(message *Message, schema *preset.Schema) error {
	pattern, err := compileRegexp(&schema.ParserPreset.HeaderPattern)
	if err != nil {
		return fmt.Errorf("compile header pattern: %w", err)
	}

	matches := pattern.FindStringSubmatch(message.Header)
	if len(matches) == 0 {
		return nil
	}

	for index, name := range schema.ParserPreset.HeaderCorrespondence {
		if index+1 >= len(matches) {
			break
		}

		switch name {
		case "type":
			message.Type = matches[index+1]
		case "scope":
			message.Scope = matches[index+1]
		case "subject":
			message.Subject = matches[index+1]
		}
	}

	return nil
}

func Lint(raw, source string, schema *preset.Schema) (Result, error) {
	message, err := Parse(raw, schema)
	if err != nil {
		return Result{}, err
	}

	evaluator := newEvaluator(schema, message)
	if err := evaluator.evaluate(); err != nil {
		return Result{}, err
	}

	return Result{
		Source:   source,
		Valid:    countByLevel(evaluator.findings, LevelError) == 0,
		Findings: evaluator.findings,
	}, nil
}

type evaluator struct {
	schema   *preset.Schema
	message  Message //nolint:ptrstruct // pointer causes message to escape to heap (+1 alloc/op); value copy is cheaper here
	findings []Finding
}

func newEvaluator(schema *preset.Schema, message Message) evaluator { //nolint:ptrstruct // pointer causes message to escape to heap (+1 alloc/op); value copy is cheaper here
	return evaluator{
		schema:   schema,
		message:  message,
		findings: make([]Finding, 0, len(schema.Rules)),
	}
}

func (e *evaluator) evaluate() error {
	steps := []func() error{
		e.evaluateHeader,
		e.evaluateSubject,
		e.evaluateType,
		e.evaluateBody,
		e.evaluateFooter,
	}

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

func (e *evaluator) evaluateHeader() error {
	trimFact, trimMsg := headerTrimCheck(e.message.Header)
	e.appendRule(preset.RuleHeaderTrim, "header", trimFact, trimMsg)

	headerMaxRule := e.schema.Rules[preset.RuleHeaderMaxLength]
	return checkMaxLength(&headerMaxRule, func(limit int) error {
		e.appendRule(
			preset.RuleHeaderMaxLength,
			"header",
			utf16Length(e.message.Header) <= limit,
			fmt.Sprintf("header exceeds max length %d", limit),
		)
		return nil
	})
}

func (e *evaluator) evaluateType() error {
	e.appendRule(
		preset.RuleTypeEmpty,
		"type",
		e.message.Type == "",
		"type may not be empty",
	)

	typeCaseRule := e.schema.Rules[preset.RuleTypeCase]
	if err := checkStringValue(&typeCaseRule, func(expected string) error {
		e.appendRule(
			preset.RuleTypeCase,
			"type",
			matchCase(e.message.Type, expected),
			fmt.Sprintf("type must be %s", expected),
		)
		return nil
	}); err != nil {
		return err
	}

	typeEnumRule := e.schema.Rules[preset.RuleTypeEnum]
	return checkStringListValue(&typeEnumRule, func(allowed []string) error {
		if e.message.Type == "" {
			return nil
		}

		e.appendRule(
			preset.RuleTypeEnum,
			"type",
			slices.Contains(allowed, e.message.Type),
			fmt.Sprintf("type must be one of: %s", strings.Join(allowed, ", ")),
		)
		return nil
	})
}

func (e *evaluator) evaluateSubject() error {
	e.appendRule(
		preset.RuleSubjectEmpty,
		"subject",
		e.message.Subject == "",
		"subject may not be empty",
	)

	subjectFullStopRule := e.schema.Rules[preset.RuleSubjectFullStop]
	if err := checkStringValue(&subjectFullStopRule, func(disallowed string) error {
		hasStop := strings.HasSuffix(e.message.Subject, disallowed) && !strings.HasSuffix(e.message.Subject, subjectEllipsis)
		e.appendRule(
			preset.RuleSubjectFullStop,
			"subject",
			hasStop,
			fmt.Sprintf("subject may not end with %q", disallowed),
		)
		return nil
	}); err != nil {
		return err
	}

	subjectCaseRule := e.schema.Rules[preset.RuleSubjectCase]
	return checkStringListValue(&subjectCaseRule, func(disallowed []string) error {
		if !startsWithCasedLetter(e.message.Subject) {
			return nil
		}

		fact := matchesAnySubjectCase(e.message.Subject, disallowed)
		e.appendRule(
			preset.RuleSubjectCase,
			"subject",
			fact,
			subjectCaseMessage(subjectCaseRule.Applicable, disallowed),
		)
		return nil
	})
}

func (e *evaluator) evaluateBody() error {
	if len(e.message.BodyLines) > 0 {
		e.appendRule(
			preset.RuleBodyLeadingBlank,
			"body",
			e.message.BodyLeadingBlank,
			"body must begin with a blank line",
		)
	}

	bodyMaxRule := e.schema.Rules[preset.RuleBodyMaxLineLength]
	return checkMaxLength(&bodyMaxRule, func(limit int) error {
		for index, line := range e.message.BodyLines {
			if line == "" || containsURL(line) {
				continue
			}

			if utf16Length(line) > limit {
				e.appendRule(
					preset.RuleBodyMaxLineLength,
					"body",
					false,
					fmt.Sprintf("body line %d exceeds max length %d", index+1, limit),
				)
				return nil
			}
		}

		return nil
	})
}

func (e *evaluator) evaluateFooter() error {
	if len(e.message.FooterLines) > 0 {
		e.appendRule(
			preset.RuleFooterLeadingBlank,
			"footer",
			e.message.FooterLeadingBlank,
			"footer must begin with a blank line",
		)
	}

	footerMaxRule := e.schema.Rules[preset.RuleFooterMaxLineLength]
	return checkMaxLength(&footerMaxRule, func(limit int) error {
		for index, line := range e.message.FooterLines {
			if line == "" {
				continue
			}

			if utf16Length(line) > limit {
				e.appendRule(
					preset.RuleFooterMaxLineLength,
					"footer",
					false,
					fmt.Sprintf("footer line %d exceeds max length %d", index+1, limit),
				)
				return nil
			}
		}

		return nil
	})
}

func (e *evaluator) appendRule(ruleName preset.RuleName, field string, fact bool, message string) {
	rule, exists := e.schema.Rules[ruleName]
	if !exists {
		return
	}

	if applyApplicable(rule.Applicable, fact) {
		return
	}

	e.findings = append(e.findings, Finding{
		Rule:    ruleName,
		Level:   levelFromRule(&rule),
		Field:   field,
		Message: message,
	})
}

func countByLevel(findings []Finding, level Level) int {
	count := 0
	for _, finding := range findings {
		if finding.Level == level {
			count++
		}
	}

	return count
}

func levelFromRule(rule *preset.Rule) Level {
	if rule.Level == 1 {
		return LevelWarning
	}

	return LevelError
}

func checkMaxLength(rule *preset.Rule, fn func(limit int) error) error {
	if len(rule.Value) == 0 {
		return nil
	}

	var limit int
	if err := json.Unmarshal(rule.Value, &limit); err != nil {
		return fmt.Errorf("decode max length: %w", err)
	}

	return fn(limit)
}

func checkStringValue(rule *preset.Rule, fn func(value string) error) error {
	if len(rule.Value) == 0 {
		return nil
	}

	var value string
	if err := json.Unmarshal(rule.Value, &value); err != nil {
		return fmt.Errorf("decode string rule: %w", err)
	}

	return fn(value)
}

func checkStringListValue(rule *preset.Rule, fn func(values []string) error) error {
	if len(rule.Value) == 0 {
		return nil
	}

	var values []string
	if err := json.Unmarshal(rule.Value, &values); err != nil {
		return fmt.Errorf("decode string list rule: %w", err)
	}

	return fn(values)
}

func findFooterStart(lines []string) int {
	for index := range lines {
		if !isFooterStart(lines[index]) {
			continue
		}

		if index == 0 || lines[index-1] == "" || isBreakingFooterLine(lines[index]) {
			return index
		}
	}

	return -1
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

var footerPattern = regexp.MustCompile(`^(?:[A-Za-z-]+(?:\([^)]+\))?!?)(?:: | #)|^(?:BREAKING CHANGE|BREAKING-CHANGE): `)

func isFooterStart(line string) bool {
	return footerPattern.MatchString(line)
}

var compiledRegexps sync.Map

func compileRegexp(value *preset.Regexp) (*regexp.Regexp, error) {
	if value.Flags != "" && value.Flags != "i" {
		return nil, fmt.Errorf("unsupported regexp flags %q", value.Flags)
	}

	key := value.Flags + ":" + value.Source
	if cached, ok := compiledRegexps.Load(key); ok {
		if re, ok := cached.(*regexp.Regexp); ok {
			return re, nil
		}
	}

	var re *regexp.Regexp
	var err error
	if value.Flags == "i" {
		re, err = regexp.Compile("(?i)" + value.Source)
	} else {
		re, err = regexp.Compile(value.Source)
	}
	if err != nil {
		return nil, err
	}

	compiledRegexps.Store(key, re)
	return re, nil
}

func utf16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func containsURL(line string) bool {
	return strings.Contains(line, "://")
}

func matchCase(value, expected string) bool {
	if value == "" {
		return true
	}

	if matcher, ok := subjectCaseMatchers[expected]; ok {
		return matcher(value)
	}

	return true
}

var subjectCaseMatchers = map[string]func(string) bool{
	"lower-case":    isLowerCase,
	"sentence-case": isSentenceCase,
	"start-case":    isStartCase,
	"pascal-case":   isPascalCase,
	"upper-case":    isUpperCase,
}

func matchesAnySubjectCase(subject string, cases []string) bool {
	for _, bucket := range cases {
		matcher, ok := subjectCaseMatchers[bucket]
		if ok && matcher(subject) {
			return true
		}
	}

	return false
}

func isLowerCase(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) && unicode.ToLower(r) != r {
			return false
		}
	}

	return true
}

func isSentenceCase(value string) bool {
	word, ok := firstWord(value)
	if !ok || len(word) == 0 {
		return false
	}

	if !unicode.IsUpper(word[0]) {
		return false
	}

	for index, r := range word {
		if index == 0 {
			continue
		}

		if unicode.IsUpper(r) {
			return false
		}
	}

	return true
}

func isStartCase(value string) bool {
	words := collectASCIIWords(value)
	if len(words) < 2 {
		return false
	}

	for _, word := range words {
		if len(word) == 0 || !isUpperASCII(word[0]) {
			return false
		}

		if slices.ContainsFunc(word[1:], isUpperASCII) {
			return false
		}
	}

	return true
}

func isPascalCase(value string) bool {
	if strings.ContainsAny(value, " -_") {
		return false
	}

	word := []rune(value)
	if len(word) == 0 || !isUpperASCII(word[0]) {
		return false
	}

	hasLower := false
	for _, r := range word[1:] {
		if !isASCIIAlphaNum(r) {
			return false
		}

		if isUpperASCII(r) {
			return true
		}

		if isLowerASCII(r) {
			hasLower = true
		}
	}

	return hasLower
}

func isUpperCase(value string) bool {
	hasLetter := false
	for _, r := range value {
		if !unicode.IsLetter(r) {
			continue
		}

		hasLetter = true
		if unicode.ToUpper(r) != r {
			return false
		}
	}

	return hasLetter
}

func firstWord(value string) ([]rune, bool) {
	var word []rune
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, r)
			continue
		}

		if len(word) > 0 {
			return word, true
		}
	}

	if len(word) == 0 {
		return nil, false
	}

	return word, true
}

func collectASCIIWords(value string) [][]rune {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return !isASCIIAlphaNum(r)
	})

	words := make([][]rune, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}

		words = append(words, []rune(field))
	}

	return words
}

func isUpperASCII(r rune) bool {
	return 'A' <= r && r <= 'Z'
}

func isLowerASCII(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func isASCIIAlphaNum(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9')
}

func joinCases(values []string) string {
	if len(values) == 0 {
		return ""
	}

	if len(values) == 1 {
		return values[0]
	}

	if len(values) == 2 {
		return values[0] + " or " + values[1]
	}

	return strings.Join(values[:len(values)-1], ", ") + ", or " + values[len(values)-1]
}

func subjectCaseMessage(applicable preset.Applicable, values []string) string {
	if applicable == preset.ApplicableNever {
		return fmt.Sprintf("subject must not be %s", joinCases(values))
	}

	return fmt.Sprintf("subject must be %s", joinCases(values))
}

func headerTrimCheck(header string) (bool, string) {
	hasLeading := len(strings.TrimLeftFunc(header, unicode.IsSpace)) < len(header)
	hasTrailing := len(strings.TrimRightFunc(header, unicode.IsSpace)) < len(header)

	switch {
	case hasLeading && hasTrailing:
		return false, "header must not be surrounded by whitespace"
	case hasLeading:
		return false, "header must not start with whitespace"
	case hasTrailing:
		return false, "header must not end with whitespace"
	default:
		return true, ""
	}
}

func applyApplicable(applicable preset.Applicable, fact bool) bool {
	if applicable == preset.ApplicableNever {
		return !fact
	}

	return fact
}

func isBreakingFooterLine(line string) bool {
	return strings.HasPrefix(line, "BREAKING CHANGE: ") || strings.HasPrefix(line, "BREAKING-CHANGE: ")
}

func startsWithCasedLetter(value string) bool {
	if value == "" {
		return false
	}

	r, _ := utf8.DecodeRuneInString(value)
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r)
}
