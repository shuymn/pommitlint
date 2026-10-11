package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type isolatedGitRepo struct {
	rootDir string
	env     []string
}

func TestIsolatedGitRepoIgnoresHookEnvironment(t *testing.T) {
	foreignDir := t.TempDir()
	foreignConfig := filepath.Join(foreignDir, "config")
	const originalConfig = "[core]\n\tbare = true\n"
	if err := os.WriteFile(foreignConfig, []byte(originalConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	// Git exports repository context to hooks; it must not escape into fixtures.
	t.Setenv("GIT_COMMON_DIR", foreignDir)
	t.Setenv("GIT_DIR", foreignDir)
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.commentChar")
	t.Setenv("GIT_CONFIG_VALUE_0", ";")

	repo := newIsolatedGitRepo(t)
	if _, err := os.Stat(filepath.Join(repo.rootDir, ".git", "config")); err != nil {
		t.Fatalf("fixture did not initialize its own repository: %v", err)
	}
	got, err := os.ReadFile(foreignConfig)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != originalConfig {
		t.Fatalf("fixture modified the inherited repository config: %s", got)
	}
	if config := repo.git(t, "config", "--list"); strings.Contains(config, "core.commentchar=") {
		t.Fatalf("fixture inherited command-line config: %s", config)
	}
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

	// Ignore hook-exported repository paths and injected config before setting
	// the fixture defaults. HOME isolation alone cannot override these values.
	env := slices.DeleteFunc(os.Environ(), func(entry string) bool {
		return strings.HasPrefix(entry, "GIT_")
	})
	repo := isolatedGitRepo{
		rootDir: rootDir,
		env: append(env,
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
