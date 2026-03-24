package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/shuymn/pommitlint/internal/cli"
)

func TestCompatDefaultIgnores(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		origin      string
		message     string
		args        []string
		wantIgnored bool
		wantExit    int
	}{
		{
			name:        "merge-branch",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Merge branch 'iss53'",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "merge-branch-with-comment-after-newline",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Merge branch 'ctrom-YarnBuild'\r\n # some comment",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "merge-tag",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Merge tag '1.1.1'",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "merge-pull-request",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Merge pull request #369",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "revert",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Revert \"docs: add recipe for linting of all commits in a PR (#36)\"\n\nThis reverts commit 1e69d542c16c2a32acfd139e32efa07a45f19111.",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "semver",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "v0.0.1",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "semver-with-chore-release",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "chore(release): 2.3.3-beta.1 [skip ci]",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "semver-with-footers",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "0.0.1\n\nSigned-off-by: Developer <example@example.com>",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "fixup",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "fixup! initial commit",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "squash",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "squash! initial commit",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "amend",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "amend! initial commit",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "bitbucket-merge",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Merged in feature/facebook-friends-sync (pull request #8)",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "azure-devops-merge",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Merged PR 123: Description here",
			wantIgnored: true,
			wantExit:    0,
		},
		{
			name:        "false-positive-merge-branch-not-at-start",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "foo bar Merge branch xxx",
			wantIgnored: false,
			wantExit:    1,
		},
		{
			name:        "false-positive-merge-tag-not-at-start",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "foo bar Merge tag '1.1.1'",
			wantIgnored: false,
			wantExit:    1,
		},
		{
			name:        "default-ignores-opt-out",
			origin:      "@commitlint/is-ignored/src/is-ignored.test.ts",
			message:     "Auto-merged develop into master",
			args:        []string{"--no-default-ignores"},
			wantIgnored: false,
			wantExit:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := []string{"lint", "--message", tt.message, "--format", "json"}
			args = append(args, tt.args...)
			result := runCommand(t, &commandInput{args: args})

			if diff := cmp.Diff(tt.wantExit, result.exitCode); diff != "" {
				t.Fatalf("%s: exitCode mismatch (-want +got):\n%s\nstdout=%q\nstderr=%q", tt.origin, diff, result.stdout, result.stderr)
			}

			var got cli.JSONReport
			if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
				t.Fatalf("%s: decode JSON report: %v", tt.origin, err)
			}

			if diff := cmp.Diff(tt.wantIgnored, got.Ignored); diff != "" {
				t.Fatalf("%s: Ignored mismatch (-want +got):\n%s", tt.origin, diff)
			}
		})
	}
}

func TestCompatEditDefaultsToGitCommitEditMsgFromRoot(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	editPath := filepath.Join(repo.rootDir, ".git", "COMMIT_EDITMSG")
	content := "feat: add parser\n\nbody\n# comment\n# ------------------------ >8 ------------------------\nshould be cut\n"
	if err := os.WriteFile(editPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write edit file: %v", err)
	}

	got := runEditDefaultCompat(t, repo.rootDir)

	if got.Ignored {
		t.Fatal("Ignored = true, want false")
	}
	if !got.Valid {
		t.Fatalf("Valid = false, want true, findings = %#v", got.Findings)
	}
	if diff := cmp.Diff("edit", got.Source); diff != "" {
		t.Fatalf("Source mismatch (-want +got):\n%s", diff)
	}
}

func TestCompatEditDefaultsToGitCommitEditMsgFromSubdirectory(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	subdir := filepath.Join(repo.rootDir, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	editPath := filepath.Join(repo.rootDir, ".git", "COMMIT_EDITMSG")
	content := "feat: add parser\n\nbody\n# comment\n# ------------------------ >8 ------------------------\nshould be cut\n"
	if err := os.WriteFile(editPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write edit file: %v", err)
	}

	got := runEditDefaultCompat(t, subdir)

	if got.Ignored {
		t.Fatal("Ignored = true, want false")
	}
	if !got.Valid {
		t.Fatalf("Valid = false, want true, findings = %#v", got.Findings)
	}
	if diff := cmp.Diff("edit", got.Source); diff != "" {
		t.Fatalf("Source mismatch (-want +got):\n%s", diff)
	}
}

func TestCompatEditSanitizeCommentsAndScissors(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	editPath := filepath.Join(repo.rootDir, ".git", "COMMIT_EDITMSG")
	content := "feat: add parser\n\nbody\n# comment\n# ------------------------ >8 ------------------------\nshould be cut\n"
	if err := os.WriteFile(editPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write edit file: %v", err)
	}

	result := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"lint", "--edit", editPath, "--format", "json"},
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stdout=%q, stderr=%q", result.exitCode, result.stdout, result.stderr)
	}

	var got cli.JSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	if got.Ignored {
		t.Fatal("Ignored = true, want false")
	}
	if !got.Valid {
		t.Fatalf("Valid = false, want true, findings = %#v", got.Findings)
	}
}

func runEditDefaultCompat(t *testing.T, workDir string) cli.JSONReport {
	t.Helper()

	result := runCommand(t, &commandInput{
		workDir: workDir,
		args:    []string{"lint", "--edit", "--format", "json"},
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stdout=%q, stderr=%q", result.exitCode, result.stdout, result.stderr)
	}

	var got cli.JSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	return got
}

func TestCompatEditUsesCoreCommentChar(t *testing.T) {
	t.Parallel()

	repo := newIsolatedGitRepo(t)
	repo.git(t, "config", "core.commentChar", "$")

	editPath := filepath.Join(repo.rootDir, ".git", "COMMIT_EDITMSG")
	content := "feat: add parser\n\nbody\n$ comment\n$ ------------------------ >8 ------------------------\nshould be cut\n"
	if err := os.WriteFile(editPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write edit file: %v", err)
	}

	result := runCommand(t, &commandInput{
		workDir: repo.rootDir,
		args:    []string{"lint", "--edit", editPath, "--format", "json"},
	})

	if result.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stdout=%q, stderr=%q", result.exitCode, result.stdout, result.stderr)
	}

	var got cli.JSONReport
	if err := json.Unmarshal([]byte(result.stdout), &got); err != nil {
		t.Fatalf("decode JSON report: %v", err)
	}

	if got.Ignored {
		t.Fatal("Ignored = true, want false")
	}
	if !got.Valid {
		t.Fatalf("Valid = false, want true, findings = %#v", got.Findings)
	}
}
