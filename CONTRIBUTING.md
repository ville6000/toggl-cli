# Contributing to toggl-cli

Bug reports and pull requests are welcome. For anything larger than a small
fix, open an [issue](https://github.com/ville6000/toggl-cli/issues) first so
the approach can be agreed on before you write the code.

## Development setup

You need Go (the version in `go.mod`), [gofumpt](https://github.com/mvdan/gofumpt)
and [golangci-lint](https://golangci-lint.run). The pinned versions are in
`mise.toml`; with [mise](https://mise.jdx.dev) installed, get them all with:

```sh
mise install
```

Then build and run from the checkout:

```sh
go build -o toggl-cli .
./toggl-cli --version
```

`--version` reports a version derived from git for local builds.

## Tests, formatting and linting

```sh
make test     # go test -v ./...
make format   # gofumpt -l -w .
make lint     # golangci-lint run ./...
```

CI runs the build, the tests and golangci-lint on every pull request, using the
same golangci-lint version as `mise.toml`. Run `make format lint test` before
pushing.

The command tests in `cmd/e2e_*_test.go` run the whole CLI against local stub
servers for the Toggl and 7pace APIs, with a temporary home directory, so they
never touch your real config or account.

## Project layout

| Path | Contents |
| --- | --- |
| `main.go` | Entry point: runs the root command and cancels on Ctrl-C |
| `cmd/` | The cobra commands, one file per command; `NewRootCmd` builds the tree |
| `internal/api/` | Toggl and 7pace API clients, their types and the project cache |
| `internal/config/` | Reading settings from the config file and environment |
| `internal/output/` | Table and duration formatting |

## Pull requests

- Keep each pull request to one change, with tests for new behaviour and bug
  fixes.
- Write commit messages and PR titles as
  [Conventional Commits](https://www.conventionalcommits.org): `feat: ...`,
  `fix: ...`, `refactor: ...`, `docs: ...`, `chore: ...`.
- Update the README when you change user-facing behaviour.

## Releasing

Releases are cut by the maintainer:

1. Publish a [GitHub release](https://github.com/ville6000/toggl-cli/releases/new)
   from `main` with a new `vX.Y.Z` tag, generating the release notes:

   ```sh
   gh release create vX.Y.Z --target main --generate-notes
   ```

2. Publishing starts the Release workflow (`.github/workflows/release.yml`),
   which builds the binaries with [GoReleaser](https://goreleaser.com) and
   attaches them, with `checksums.txt`, to the release.

To check the GoReleaser build before releasing, run
`goreleaser release --snapshot --clean`; the archives land in `dist/`.
