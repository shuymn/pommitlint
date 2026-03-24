package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
)

var ErrLintFailed = errors.New("lint failed")

type Options struct {
	Args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type JSONReport struct {
	Source       string        `json:"source"`
	Valid        bool          `json:"valid"`
	ErrorCount   int           `json:"errorCount"`
	WarningCount int           `json:"warningCount"`
	Findings     []JSONFinding `json:"findings"`
}

type JSONFinding struct {
	Rule    string `json:"rule"`
	Level   string `json:"level"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func Run(ctx context.Context, options *Options) (int, error) {
	app := newApp(options)
	command := newRootCommand(app)
	command.SetArgs(options.Args)

	err := command.ExecuteContext(ctx)
	if err == nil {
		return app.exitCode, nil
	}

	if errors.Is(err, ErrLintFailed) {
		return app.exitCode, ErrLintFailed
	}

	return 2, err
}

type app struct {
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	exitCode int
}

func newApp(options *Options) *app {
	return &app{
		stdin:    defaultReader(options.Stdin, os.Stdin),
		stdout:   defaultWriter(options.Stdout, os.Stdout),
		stderr:   defaultWriter(options.Stderr, os.Stderr),
		exitCode: 0,
	}
}

func newRootCommand(app *app) *cobra.Command {
	command := &cobra.Command{
		Use:           "pommitlint",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	command.SetIn(app.stdin)
	command.SetOut(app.stdout)
	command.SetErr(app.stderr)
	command.AddCommand(newLintCommand(app))

	return command
}

func newLintCommand(app *app) *cobra.Command {
	var message string
	var filePath string
	var editPath string
	var format string

	command := &cobra.Command{
		Use:   "lint",
		Short: "Lint a commit message",
		RunE: func(_ *cobra.Command, _ []string) error {
			schema, err := preset.Load()
			if err != nil {
				return err
			}

			input, source, err := resolveInput(app.stdin, message, filePath, editPath)
			if err != nil {
				return err
			}

			result, err := lint.Lint(input, source, &schema)
			if err != nil {
				return err
			}

			if err := writeReport(app.stdout, &result, format); err != nil {
				return err
			}

			if result.ErrorCount() > 0 {
				app.exitCode = 1
				return ErrLintFailed
			}

			app.exitCode = 0
			return nil
		},
	}

	command.Flags().StringVar(&message, "message", "", "lint the provided message")
	command.Flags().StringVar(&filePath, "file", "", "lint the provided file")
	command.Flags().StringVar(&editPath, "edit", "", "lint the provided edit file")
	command.Flags().StringVar(&format, "format", "text", "report format: text or json")

	return command
}

func resolveInput(stdin io.Reader, message, filePath, editPath string) (string, string, error) {
	sourceCount := 0
	if message != "" {
		sourceCount++
	}

	if filePath != "" {
		sourceCount++
	}

	if editPath != "" {
		sourceCount++
	}

	if sourceCount > 1 {
		return "", "", errors.New("exactly one of --message, --file, or --edit may be set")
	}

	switch {
	case message != "":
		return message, "message", nil
	case filePath != "":
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", "", fmt.Errorf("read --file: %w", err)
		}

		return string(content), "file", nil
	case editPath != "":
		content, err := os.ReadFile(editPath)
		if err != nil {
			return "", "", fmt.Errorf("read --edit: %w", err)
		}

		return string(content), "edit", nil
	default:
		content, err := io.ReadAll(stdin)
		if err != nil {
			return "", "", fmt.Errorf("read stdin: %w", err)
		}

		return string(content), "stdin", nil
	}
}

func defaultReader(current io.Reader, fallback *os.File) io.Reader {
	if current != nil {
		return current
	}

	return fallback
}

func defaultWriter(current io.Writer, fallback *os.File) io.Writer {
	if current != nil {
		return current
	}

	return fallback
}

func writeReport(writer io.Writer, result *lint.Result, format string) error {
	switch format {
	case "json":
		return writeJSONReport(writer, result)
	case "text":
		_, err := fmt.Fprint(writer, formatTextReport(result))
		return err
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func formatTextReport(result *lint.Result) string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "input: %s\nstatus: ", result.Source)
	if result.Valid {
		builder.WriteString("valid")
	} else {
		builder.WriteString("invalid")
	}
	_, _ = fmt.Fprintf(&builder, "\nerrors: %d\nwarnings: %d\n", result.ErrorCount(), result.WarningCount())
	if len(result.Findings) == 0 {
		return builder.String()
	}

	builder.WriteString("\n")
	for _, finding := range result.Findings {
		builder.WriteString(string(finding.Level))
		builder.WriteString(": ")
		builder.WriteString(string(finding.Rule))
		builder.WriteString(" ")
		builder.WriteString(finding.Message)
		builder.WriteString("\n")
	}

	return builder.String()
}
