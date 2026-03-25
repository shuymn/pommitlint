package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shuymn/pommitlint/internal/lint"
	"github.com/shuymn/pommitlint/internal/preset"
	"github.com/shuymn/pommitlint/internal/version"
)

var ErrLintFailed = errors.New("lint failed")

type Options struct {
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	WorkDir string
}

type jsonReport struct {
	Source       string        `json:"source"`
	Valid        bool          `json:"valid"`
	Ignored      bool          `json:"ignored"`
	ErrorCount   int           `json:"errorCount"`
	WarningCount int           `json:"warningCount"`
	Findings     []jsonFinding `json:"findings"`
}

type jsonFinding struct {
	Rule    string `json:"rule"`
	Level   string `json:"level"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

const (
	defaultEditSentinel = "__pommitlint_default_edit__"
	commitMsgHookBody   = "#!/bin/sh\nexec pommitlint lint --edit \"$1\"\n"
	scissorsSuffix      = " ------------------------ >8 ------------------------"
)

func Run(ctx context.Context, options *Options) (int, error) {
	app := newApp(options)
	command := newRootCommand(ctx, app)
	command.SetArgs(normalizeArgs(options.Args))

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
	workDir  string
}

func newApp(options *Options) *app {
	return &app{
		stdin:    defaultReader(options.Stdin, os.Stdin),
		stdout:   defaultWriter(options.Stdout, os.Stdout),
		stderr:   defaultWriter(options.Stderr, os.Stderr),
		exitCode: 0,
		workDir:  options.WorkDir,
	}
}

func newRootCommand(ctx context.Context, app *app) *cobra.Command {
	command := &cobra.Command{
		Use:           "pommitlint",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	command.SetIn(app.stdin)
	command.SetOut(app.stdout)
	command.SetErr(app.stderr)
	command.AddCommand(newLintCommand(ctx, app))
	command.AddCommand(newHookCommand(ctx, app))

	return command
}

func newLintCommand(ctx context.Context, app *app) *cobra.Command {
	var message string
	var filePath string
	var editPath string
	var format string
	var noDefaultIgnores bool

	command := &cobra.Command{
		Use:   "lint",
		Short: "Lint a commit message",
		RunE: func(_ *cobra.Command, _ []string) error {
			schema, err := preset.Load()
			if err != nil {
				return err
			}

			input, source, err := resolveInput(ctx, app.stdin, app.workDir, message, filePath, editPath)
			if err != nil {
				return err
			}

			if !noDefaultIgnores && shouldIgnore(input) {
				result := lint.Result{
					Source:   source,
					Valid:    true,
					Ignored:  true,
					Findings: []lint.Finding{},
				}
				app.exitCode = 0
				return writeReport(app.stdout, &result, format)
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
	command.Flags().BoolVar(&noDefaultIgnores, "no-default-ignores", false, "disable built-in ignore rules")

	return command
}

func newHookCommand(ctx context.Context, app *app) *cobra.Command {
	var hooksDir string
	var force bool

	command := &cobra.Command{
		Use:   "hook",
		Short: "Manage git hooks",
	}

	installCommand := &cobra.Command{
		Use:   "install",
		Short: "Install the commit-msg hook",
		RunE: func(_ *cobra.Command, _ []string) error {
			targetPath, err := resolveHookPath(ctx, app.workDir, hooksDir)
			if err != nil {
				return err
			}

			if err := writeHook(targetPath, force); err != nil {
				return err
			}

			app.exitCode = 0
			return nil
		},
	}
	installCommand.Flags().StringVar(&hooksDir, "hooks-dir", "", "override the hooks directory")
	installCommand.Flags().BoolVar(&force, "force", false, "overwrite an existing hook")
	command.AddCommand(installCommand)

	return command
}

func resolveInput(ctx context.Context, stdin io.Reader, workDir, message, filePath, editPath string) (string, string, error) {
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
		resolvedEditPath := editPath
		if resolvedEditPath == defaultEditSentinel {
			resolvedEditPath = resolveDefaultEditPath(ctx, workDir)
		}

		content, err := os.ReadFile(resolvedEditPath)
		if err != nil {
			return "", "", fmt.Errorf("read --edit: %w", err)
		}

		return sanitizeEditMessage(ctx, string(content), resolvedEditPath, workDir), "edit", nil
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
	switch {
	case result.Ignored:
		builder.WriteString("ignored")
	case result.Valid:
		builder.WriteString("valid")
	default:
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

func normalizeArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		current := args[index]
		if current != "--edit" {
			normalized = append(normalized, current)
			continue
		}

		if index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
			normalized = append(normalized, current, args[index+1])
			index++
			continue
		}

		normalized = append(normalized, "--edit="+defaultEditSentinel)
	}

	return normalized
}

var defaultIgnorePatterns = []func(string) bool{
	regexp.MustCompile(`(?m)^((Merge pull request)|(Merge (.*?) into (.*?)|(Merge branch (.*?)))(?:\r?\n)*$)`).MatchString,
	regexp.MustCompile(`(?m)^(Merge tag (.*?))(?:\r?\n)*$`).MatchString,
	regexp.MustCompile(`^(R|r)evert (.*)`).MatchString,
	regexp.MustCompile(`^(R|r)eapply (.*)`).MatchString,
	regexp.MustCompile(`^(amend|fixup|squash)!`).MatchString,
	isSemverMessage,
	regexp.MustCompile(`^(Merged (.*?)(in|into) (.*)|Merged PR (.*): (.*))`).MatchString,
	regexp.MustCompile(`^Merge remote-tracking branch(\s*)(.*)`).MatchString,
	regexp.MustCompile(`^Automatic merge(.*)`).MatchString,
	regexp.MustCompile(`^Auto-merged (.*?) into (.*)`).MatchString,
}

var (
	semverPattern            = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	semverChorePrefixPattern = regexp.MustCompile(`^chore(\([^)]+\))?:`)
)

func shouldIgnore(message string) bool {
	for _, pattern := range defaultIgnorePatterns {
		if pattern(message) {
			return true
		}
	}

	return false
}

func isSemverMessage(message string) bool {
	firstLine, _, _ := strings.Cut(message, "\n")
	stripped := semverChorePrefixPattern.ReplaceAllString(firstLine, "")
	return semverPattern.MatchString(strings.TrimSpace(stripped))
}

func sanitizeEditMessage(ctx context.Context, raw, editPath, workDir string) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	sanitizeDir := resolveEditSanitizeDir(ctx, editPath, workDir)
	prefix, ok := resolveCommentPrefix(ctx, normalized, sanitizeDir)
	if ok {
		scissorsLine := prefix + scissorsSuffix
		if head, _, found := strings.Cut(normalized, scissorsLine+"\n"); found {
			normalized = head
		} else if head, _, found := strings.Cut(normalized, scissorsLine); found {
			normalized = head
		}

		if stripped, stripOK := gitStripspace(ctx, normalized, sanitizeDir, "--strip-comments"); stripOK {
			return strings.TrimRight(stripped, "\n")
		}
	}

	return strings.TrimRight(fallbackSanitizeEditMessage(normalized, "#"), "\n")
}

func resolveCommentPrefix(ctx context.Context, raw, workDir string) (string, bool) {
	const sentinel = "pommitlint-comment-prefix-sentinel"
	commented, ok := gitStripspace(ctx, raw+"\n"+sentinel+"\n", workDir, "--comment-lines")
	if !ok {
		return "", false
	}

	for line := range strings.SplitSeq(strings.TrimRight(commented, "\n"), "\n") {
		if prefix, ok := strings.CutSuffix(line, sentinel); ok {
			return strings.TrimSuffix(prefix, " "), true
		}
	}

	return "", false
}

func gitStripspace(ctx context.Context, input, workDir, mode string) (string, bool) {
	command := exec.CommandContext(ctx, "git", "stripspace", mode)
	command.Dir = defaultWorkDir(workDir)
	command.Stdin = strings.NewReader(input)
	output, err := command.Output()
	if err != nil {
		return "", false
	}

	return string(output), true
}

func resolveEditSanitizeDir(ctx context.Context, editPath, workDir string) string {
	if editPath == "" {
		return defaultWorkDir(workDir)
	}

	editDir := filepath.Dir(editPath)
	command := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	command.Dir = editDir
	output, err := command.Output()
	if err != nil {
		return editDir
	}

	return strings.TrimSpace(string(output))
}

func resolveDefaultEditPath(ctx context.Context, workDir string) string {
	baseDir := defaultWorkDir(workDir)
	resolved, err := resolveGitPath(ctx, baseDir, "COMMIT_EDITMSG")
	if err != nil {
		return filepath.Join(baseDir, ".git", "COMMIT_EDITMSG")
	}
	return resolved
}

func resolveGitPath(ctx context.Context, baseDir, gitPathArg string) (string, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--git-path", gitPathArg)
	command.Dir = baseDir
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	resolved := strings.TrimSpace(string(output))
	if filepath.IsAbs(resolved) {
		return resolved, nil
	}
	return filepath.Join(baseDir, resolved), nil
}

func fallbackSanitizeEditMessage(raw, commentPrefix string) string {
	lines := strings.Split(raw, "\n")
	sanitized := make([]string, 0, len(lines))
	scissorsLine := commentPrefix + scissorsSuffix

	for _, line := range lines {
		if line == scissorsLine {
			break
		}
		if strings.HasPrefix(line, commentPrefix) {
			continue
		}

		sanitized = append(sanitized, line)
	}

	return strings.Join(sanitized, "\n")
}

func resolveHookPath(ctx context.Context, workDir, hooksDir string) (string, error) {
	baseDir := defaultWorkDir(workDir)
	if hooksDir != "" {
		return filepath.Join(hooksDir, "commit-msg"), nil
	}

	hooksPath, err := resolveGitPath(ctx, baseDir, "hooks")
	if err != nil {
		return "", fmt.Errorf("resolve git hooks path: %w", err)
	}

	return filepath.Join(hooksPath, "commit-msg"), nil
}

func writeHook(targetPath string, force bool) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create hook directory: %w", err)
	}

	info, err := os.Lstat(targetPath)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("refusing to overwrite symlink hook at %s", targetPath)
	case err == nil && !force:
		return fmt.Errorf("hook already exists at %s", targetPath)
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stat hook: %w", err)
	}

	if err := os.WriteFile(targetPath, []byte(commitMsgHookBody), 0o600); err != nil {
		return fmt.Errorf("write hook: %w", err)
	}

	if err := os.Chmod(targetPath, 0o755); err != nil {
		return fmt.Errorf("chmod hook: %w", err)
	}

	return nil
}

func defaultWorkDir(workDir string) string {
	if workDir != "" {
		return workDir
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	return cwd
}
