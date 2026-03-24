package preset

import "encoding/json"

const SchemaVersion = 1

type RuleName string

const (
	RuleBodyLeadingBlank    RuleName = "body-leading-blank"
	RuleBodyMaxLineLength   RuleName = "body-max-line-length"
	RuleFooterLeadingBlank  RuleName = "footer-leading-blank"
	RuleFooterMaxLineLength RuleName = "footer-max-line-length"
	RuleHeaderMaxLength     RuleName = "header-max-length"
	RuleHeaderTrim          RuleName = "header-trim"
	RuleSubjectCase         RuleName = "subject-case"
	RuleSubjectEmpty        RuleName = "subject-empty"
	RuleSubjectFullStop     RuleName = "subject-full-stop"
	RuleTypeCase            RuleName = "type-case"
	RuleTypeEmpty           RuleName = "type-empty"
	RuleTypeEnum            RuleName = "type-enum"
)

type Applicable string

const (
	ApplicableAlways Applicable = "always"
	ApplicableNever  Applicable = "never"
)

type Schema struct {
	Version      int               `json:"version"`
	Source       Source            `json:"source"` //nolint:ptrstruct // public API field: pointer changes package contract and forces nil checks at all access sites
	Rules        map[RuleName]Rule `json:"rules"`
	ParserPreset ParserPreset      `json:"parserPreset"`
}

type Source struct {
	ConfigPackage       string `json:"configPackage"`
	ParserPresetPackage string `json:"parserPresetPackage"`
}

type Rule struct {
	Level      int             `json:"level"`
	Applicable Applicable      `json:"applicable"`
	Value      json.RawMessage `json:"value,omitempty"`
}

type ParserPreset struct {
	Name                  string   `json:"name"`
	HeaderPattern         Regexp   `json:"headerPattern"` //nolint:ptrstruct // public API field: pointer changes package contract and forces nil checks at all access sites
	BreakingHeaderPattern Regexp   `json:"breakingHeaderPattern"`
	HeaderCorrespondence  []string `json:"headerCorrespondence"`
	NoteKeywords          []string `json:"noteKeywords"`
	RevertPattern         Regexp   `json:"revertPattern"`
	RevertCorrespondence  []string `json:"revertCorrespondence"`
	IssuePrefixes         []string `json:"issuePrefixes"`
}

type Regexp struct {
	Source string `json:"source"`
	Flags  string `json:"flags"`
}

func KnownRules() map[RuleName]struct{} {
	return map[RuleName]struct{}{
		RuleBodyLeadingBlank:    {},
		RuleBodyMaxLineLength:   {},
		RuleFooterLeadingBlank:  {},
		RuleFooterMaxLineLength: {},
		RuleHeaderMaxLength:     {},
		RuleHeaderTrim:          {},
		RuleSubjectCase:         {},
		RuleSubjectEmpty:        {},
		RuleSubjectFullStop:     {},
		RuleTypeCase:            {},
		RuleTypeEmpty:           {},
		RuleTypeEnum:            {},
	}
}
