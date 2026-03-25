package main

import (
	"slices"
	"strings"
)

type CorpusEntry struct {
	ID      string   `json:"id"`
	Message string   `json:"message"`
	Tags    []string `json:"tags"`
}

// findingSummary uses only rule name and level for comparison.
// Message text is excluded because wording legitimately differs between implementations.
type findingSummary struct {
	Rule  string `json:"rule"`
	Level string `json:"level"`
}

type lintResult struct {
	ID       string           `json:"id"`
	Valid    bool             `json:"valid"`
	Ignored  bool             `json:"ignored"`
	Findings []findingSummary `json:"findings"`
}

type comparisonDiff struct {
	ID                string           `json:"id"`
	Message           string           `json:"message"`
	CommitlintValid   bool             `json:"commitlintValid"`
	PommitlintValid   bool             `json:"pommitlintValid"`
	CommitlintIgnored bool             `json:"commitlintIgnored"`
	PommitlintIgnored bool             `json:"pommitlintIgnored"`
	OnlyCommitlint    []findingSummary `json:"onlyCommitlint,omitempty"`
	OnlyPommitlint    []findingSummary `json:"onlyPommitlint,omitempty"`
}

func Compare(commitlint, pommitlint lintResult, message string) *comparisonDiff {
	if commitlint.Valid == pommitlint.Valid &&
		commitlint.Ignored == pommitlint.Ignored &&
		findingsEqual(commitlint.Findings, pommitlint.Findings) {
		return nil
	}

	onlyCL, onlyPL := findingsDiff(commitlint.Findings, pommitlint.Findings)

	return &comparisonDiff{
		ID:                commitlint.ID,
		Message:           message,
		CommitlintValid:   commitlint.Valid,
		PommitlintValid:   pommitlint.Valid,
		CommitlintIgnored: commitlint.Ignored,
		PommitlintIgnored: pommitlint.Ignored,
		OnlyCommitlint:    onlyCL,
		OnlyPommitlint:    onlyPL,
	}
}

func findingsEqual(a, b []findingSummary) bool {
	return slices.Equal(sortedFindings(a), sortedFindings(b))
}

func findingsDiff(commitlint, pommitlint []findingSummary) (onlyCL, onlyPL []findingSummary) {
	clSet := toSet(commitlint)
	plSet := toSet(pommitlint)

	for k := range clSet {
		if !plSet[k] {
			onlyCL = append(onlyCL, k)
		}
	}

	for k := range plSet {
		if !clSet[k] {
			onlyPL = append(onlyPL, k)
		}
	}

	slices.SortFunc(onlyCL, cmpFinding)
	slices.SortFunc(onlyPL, cmpFinding)

	return onlyCL, onlyPL
}

func toSet(findings []findingSummary) map[findingSummary]bool {
	s := make(map[findingSummary]bool, len(findings))
	for _, f := range findings {
		s[f] = true
	}
	return s
}

func sortedFindings(findings []findingSummary) []findingSummary {
	sorted := make([]findingSummary, len(findings))
	copy(sorted, findings)
	slices.SortFunc(sorted, cmpFinding)
	return sorted
}

func cmpFinding(a, b findingSummary) int {
	if c := strings.Compare(a.Rule, b.Rule); c != 0 {
		return c
	}
	return strings.Compare(a.Level, b.Level)
}
