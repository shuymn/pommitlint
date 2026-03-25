package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/shuymn/pommitlint/internal/cli"
)

func TestEditMode(t *testing.T) {
	tests := []struct {
		name      string
		configKey string
		configVal string
		comment   string
		scissors  string
	}{
		{
			name:      "commentChar",
			configKey: "core.commentChar",
			configVal: ";",
			comment:   "; Please enter the commit message",
			scissors:  "; ------------------------ >8 ------------------------",
		},
		{
			name:      "commentString",
			configKey: "core.commentString",
			configVal: "//",
			comment:   "// Please enter the commit message",
			scissors:  "// ------------------------ >8 ------------------------",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newIsolatedGitRepo(t)
			repo.git(t, "config", tt.configKey, tt.configVal)

			gitDir := filepath.Join(repo.rootDir, ".git")
			editPath := filepath.Join(gitDir, "COMMIT_EDITMSG")
			message := "feat: add parser\n\nbody\n\n" + tt.comment + "\n" + tt.scissors + "\n" +
				"wrap this line because it should fail the body length rule once it is long enough to exceed one hundred characters\n"
			if err := os.WriteFile(editPath, []byte(message), 0o600); err != nil {
				t.Fatalf("write edit file: %v", err)
			}

			result := runCommand(t, &commandInput{
				workDir: repo.rootDir,
				args:    []string{"lint", "--edit", "--format", "json"},
			})
			resultFromOutside := runCommand(t, &commandInput{
				workDir: t.TempDir(),
				args:    []string{"lint", "--edit", editPath, "--format", "json"},
			})

			if result.exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0", result.exitCode)
			}
			if resultFromOutside.exitCode != 0 {
				t.Fatalf("outside exitCode = %d, want 0", resultFromOutside.exitCode)
			}

			var got cli.ExportJSONReport
			if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
				t.Fatalf("decode JSON report: %v", err)
			}
			var gotOutside cli.ExportJSONReport
			if err := json.Unmarshal([]byte(resultFromOutside.stdout), &gotOutside); err != nil {
				t.Fatalf("decode outside JSON report: %v", err)
			}

			want := cli.ExportJSONReport{
				Source:       "edit",
				Valid:        true,
				Ignored:      false,
				ErrorCount:   0,
				WarningCount: 0,
				Findings:     []cli.ExportJSONFinding{},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("report mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(want, gotOutside); diff != "" {
				t.Fatalf("outside report mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIgnore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
	}{
		{name: "merge", message: "Merge branch 'main' into feature"},
		{name: "revert", message: "Revert \"feat: add parser\""},
		{name: "semver", message: "v1.2.3"},
		{name: "auto-merge", message: "Automatic merge from main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := runCommand(t, &commandInput{
				args: []string{"lint", "--message", tt.message, "--format", "json"},
			})

			if result.exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0", result.exitCode)
			}

			var got cli.ExportJSONReport
			if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
				t.Fatalf("decode JSON report: %v", err)
			}

			want := cli.ExportJSONReport{
				Source:       "message",
				Valid:        true,
				Ignored:      true,
				ErrorCount:   0,
				WarningCount: 0,
				Findings:     []cli.ExportJSONFinding{},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("report mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIgnoreOptOut(t *testing.T) {
	t.Parallel()

	result := runCommand(t, &commandInput{
		args: []string{
			"lint",
			"--message", "Merge branch 'main' into feature",
			"--no-default-ignores",
			"--format", "json",
		},
	})

	if result.exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", result.exitCode)
	}

	var got cli.ExportJSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	if got.Ignored {
		t.Fatal("Ignored = true, want false")
	}
	if got.Valid {
		t.Fatal("Valid = true, want false")
	}
	if got.ErrorCount == 0 {
		t.Fatal("ErrorCount = 0, want > 0")
	}
}

func TestHookInstall(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)

	result := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"hook", "install"},
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", result.exitCode, result.stderr)
	}

	hookPath := filepath.Join(repo.rootDir, ".git", "hooks", "commit-msg")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook file: %v", err)
	}

	if diff := cmp.Diff("#!/bin/sh\nexec pommitlint lint --edit \"$1\"\n", string(content)); diff != "" {
		t.Fatalf("hook content mismatch (-want +got):\n%s", diff)
	}

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat hook file: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestHookInstallCoreHooksPathReplay(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	hooksDir := filepath.Join(repo.rootDir, "custom-hooks")
	repo.git(t, "config", "core.hooksPath", hooksDir)

	result := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"hook", "install"},
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", result.exitCode, result.stderr)
	}

	if _, err := os.Stat(filepath.Join(hooksDir, "commit-msg")); err != nil {
		t.Fatalf("stat custom hook: %v", err)
	}
}

func TestHookInstallForceGuardReplay(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	hookPath := filepath.Join(repo.rootDir, ".git", "hooks", "commit-msg")
	original := "#!/bin/sh\nexit 42\n"
	if err := os.WriteFile(hookPath, []byte(original), 0o600); err != nil {
		t.Fatalf("write existing hook: %v", err)
	}
	if err := os.Chmod(hookPath, 0o755); err != nil {
		t.Fatalf("chmod existing hook: %v", err)
	}

	result := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"hook", "install"},
	})

	if result.exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", result.exitCode)
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook file: %v", err)
	}
	if diff := cmp.Diff(original, string(content)); diff != "" {
		t.Fatalf("hook content mismatch (-want +got):\n%s", diff)
	}

	forced := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"hook", "install", "--force"},
	})
	if forced.exitCode != 0 {
		t.Fatalf("forced exitCode = %d, want 0, stderr=%q", forced.exitCode, forced.stderr)
	}
}

func TestHookInstallSymlinkGuardReplay(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	outsidePath := filepath.Join(t.TempDir(), "outside-hook")
	if err := os.WriteFile(outsidePath, []byte("outside\n"), 0o600); err != nil {
		t.Fatalf("write outside target: %v", err)
	}

	hookPath := filepath.Join(repo.rootDir, ".git", "hooks", "commit-msg")
	if err := os.Symlink(outsidePath, hookPath); err != nil {
		t.Fatalf("symlink hook: %v", err)
	}

	result := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"hook", "install", "--force"},
	})

	if result.exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", result.exitCode)
	}

	content, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside target: %v", err)
	}
	if diff := cmp.Diff("outside\n", string(content)); diff != "" {
		t.Fatalf("outside target mismatch (-want +got):\n%s", diff)
	}
}

func TestGitIsolationNoSignReplay(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	repo.git(t, "config", "commit.gpgsign", "true")
	repo.writeFile(t, "tracked.txt", "payload\n")
	repo.git(t, "add", "tracked.txt")
	repo.commit(t, "feat: isolated commit")
}

type commandInput struct {
	workDir string
	stdin   string
	args    []string
}

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runCommand(t *testing.T, input *commandInput) commandResult {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	options := cli.Options{
		Args:    input.args,
		Stdin:   bytes.NewBufferString(input.stdin),
		Stdout:  &stdout,
		Stderr:  &stderr,
		WorkDir: input.workDir,
	}

	exitCode, err := cli.Run(t.Context(), &options)
	_ = err // errors are captured via exitCode; callers assert on output

	return commandResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
}
