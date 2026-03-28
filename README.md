# pommitlint

Single-binary commit message linter compatible with [`@commitlint/config-conventional`](https://github.com/conventional-changelog/commitlint/tree/master/%40commitlint/config-conventional). No Node.js runtime required.

## Install

```bash
brew install shuymn/tap/pommitlint
# or
go install github.com/shuymn/pommitlint@latest
```

## Usage

### Lint a commit message

```bash
# from stdin
echo "feat(auth): add login endpoint" | pommitlint lint

# from a string
pommitlint lint --message "fix: resolve nil pointer"

# from a file
pommitlint lint --file COMMIT_MSG.txt

# from the Git edit file (used by hooks)
pommitlint lint --edit
```

`--format json` switches to JSON output:

```bash
pommitlint lint --format json --message "feat: add feature"
```

```json
{
  "source": "message",
  "valid": true,
  "ignored": false,
  "errorCount": 0,
  "warningCount": 0,
  "findings": []
}
```

### Install a Git hook

```bash
pommitlint hook install
```

This writes a `commit-msg` hook that runs `pommitlint lint --edit "$1"`. Use `--force` to overwrite an existing hook, or `--hooks-dir` to specify a custom hooks directory.

### Default ignores

Merge commits, reverts, fixups, squashes, and version tags are ignored by default. Pass `--no-default-ignores` to disable this behavior.

## Rules

pommitlint enforces the following rules from `@commitlint/config-conventional`:

| Rule | Description |
|---|---|
| `body-leading-blank` | Body must begin with a blank line |
| `body-max-line-length` | Body lines must not exceed 100 characters |
| `footer-leading-blank` | Footer must begin with a blank line |
| `footer-max-line-length` | Footer lines must not exceed 100 characters |
| `header-max-length` | Header must not exceed 100 characters |
| `header-trim` | Header must not have leading/trailing whitespace |
| `subject-case` | Subject must not be sentence-case, start-case, pascal-case, or upper-case |
| `subject-empty` | Subject must not be empty |
| `subject-full-stop` | Subject must not end with a period |
| `type-case` | Type must be lower-case |
| `type-empty` | Type must not be empty |
| `type-enum` | Type must be one of: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test` |

## Development

Requires [Task](https://taskfile.dev/).

```bash
task build     # build the binary
task test      # run tests with race detection
task lint      # run linter
task check     # lint + build + test
```

## License

[MIT](LICENSE)

This project embeds data derived from commitlint packages. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for details.
