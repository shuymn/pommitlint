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

// FindingSummary uses only rule name and level for comparison.
// Message text is excluded because wording legitimately differs between implementations.
type FindingSummary struct {
	Rule  string `json:"rule"`
	Level string `json:"level"`
}

type LintResult struct {
	ID       string           `json:"id"`
	Valid    bool             `json:"valid"`
	Ignored  bool             `json:"ignored"`
	Findings []FindingSummary `json:"findings"`
}

type Diff struct {
	ID                string           `json:"id"`
	Message           string           `json:"message"`
	CommitlintValid   bool             `json:"commitlintValid"`
	PommitlintValid   bool             `json:"pommitlintValid"`
	CommitlintIgnored bool             `json:"commitlintIgnored"`
	PommitlintIgnored bool             `json:"pommitlintIgnored"`
	OnlyCommitlint    []FindingSummary `json:"onlyCommitlint,omitempty"`
	OnlyPommitlint    []FindingSummary `json:"onlyPommitlint,omitempty"`
}

func Compare(commitlint, pommitlint LintResult, message string) *Diff {
	if commitlint.Valid == pommitlint.Valid &&
		commitlint.Ignored == pommitlint.Ignored &&
		findingsEqual(commitlint.Findings, pommitlint.Findings) {
		return nil
	}

	onlyCL, onlyPL := findingsDiff(commitlint.Findings, pommitlint.Findings)

	return &Diff{
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

func findingsEqual(a, b []FindingSummary) bool {
	return slices.Equal(sortedFindings(a), sortedFindings(b))
}

func findingsDiff(commitlint, pommitlint []FindingSummary) (onlyCL, onlyPL []FindingSummary) {
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

func toSet(findings []FindingSummary) map[FindingSummary]bool {
	s := make(map[FindingSummary]bool, len(findings))
	for _, f := range findings {
		s[f] = true
	}
	return s
}

func sortedFindings(findings []FindingSummary) []FindingSummary {
	sorted := make([]FindingSummary, len(findings))
	copy(sorted, findings)
	slices.SortFunc(sorted, cmpFinding)
	return sorted
}

func cmpFinding(a, b FindingSummary) int {
	if c := strings.Compare(a.Rule, b.Rule); c != 0 {
		return c
	}
	return strings.Compare(a.Level, b.Level)
}
