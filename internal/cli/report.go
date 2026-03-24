package cli

import (
	"encoding/json"
	"io"

	"github.com/shuymn/pommitlint/internal/lint"
)

func writeJSONReport(writer io.Writer, result *lint.Result) error {
	report := JSONReport{
		Source:       result.Source,
		Valid:        result.Valid,
		Ignored:      result.Ignored,
		ErrorCount:   result.ErrorCount(),
		WarningCount: result.WarningCount(),
		Findings:     make([]JSONFinding, 0, len(result.Findings)),
	}

	for _, finding := range result.Findings {
		report.Findings = append(report.Findings, JSONFinding{
			Rule:    string(finding.Rule),
			Level:   string(finding.Level),
			Field:   finding.Field,
			Message: finding.Message,
		})
	}

	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(true)

	return encoder.Encode(report)
}
