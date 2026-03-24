package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type isolatedGitRepo struct {
	rootDir string
	env     []string
}

func newIsolatedGitRepo(t *testing.T) isolatedGitRepo {
	t.Helper()

	rootDir := t.TempDir()
	homeDir := filepath.Join(rootDir, "home")
	xdgDir := filepath.Join(rootDir, "xdg")
	gnupgDir := filepath.Join(rootDir, "gnupg")
	hooksDir := filepath.Join(rootDir, "hooks-empty")

	for _, dir := range []string{homeDir, xdgDir, gnupgDir, hooksDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	repo := isolatedGitRepo{
		rootDir: rootDir,
		env: append(os.Environ(),
			"HOME="+homeDir,
			"XDG_CONFIG_HOME="+xdgDir,
			"GNUPGHOME="+gnupgDir,
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_TERMINAL_PROMPT=0",
			"GIT_ASKPASS=/bin/false",
			"SSH_ASKPASS=/bin/false",
			"GCM_INTERACTIVE=Never",
		),
	}

	repo.git(t, "-c", "init.defaultBranch=main", "init")
	repo.git(t, "config", "user.name", "pommitlint")
	repo.git(t, "config", "user.email", "pommitlint@example.com")

	return repo
}

func (r *isolatedGitRepo) git(t *testing.T, args ...string) string {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = r.rootDir
	command.Env = r.env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}

	return string(output)
}

func (r *isolatedGitRepo) commit(t *testing.T, message string) {
	t.Helper()
	r.git(t, "config", "core.hooksPath", filepath.Join(r.rootDir, "hooks-empty"))
	r.git(t, "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", "commit", "-m", message)
}

func (r *isolatedGitRepo) writeFile(t *testing.T, name, content string) {
	t.Helper()

	path := filepath.Join(r.rootDir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
