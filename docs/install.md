# Installation

## Homebrew

On macOS and Linux:

```sh
brew install ville6000/tap/toggl-cli
```

## Prebuilt binaries

Each [release](https://github.com/ville6000/toggl-cli/releases) has archives for
macOS, Linux and Windows on amd64 and arm64, plus a `checksums.txt`.

On macOS and Linux, set `PLATFORM` to `darwin_arm64` (Apple silicon),
`darwin_amd64` (Intel Mac), `linux_amd64` or `linux_arm64`:

```sh
PLATFORM=darwin_arm64
BASE=https://github.com/ville6000/toggl-cli/releases/latest/download
curl -fsSLO "$BASE/toggl-cli_$PLATFORM.tar.gz"
curl -fsSLO "$BASE/checksums.txt"
grep "toggl-cli_$PLATFORM.tar.gz" checksums.txt | shasum -a 256 -c
gh attestation verify "toggl-cli_$PLATFORM.tar.gz" --repo ville6000/toggl-cli
tar -xzf "toggl-cli_$PLATFORM.tar.gz" toggl-cli
sudo mv toggl-cli /usr/local/bin/
```

On Windows, download `toggl-cli_windows_amd64.zip` (or `_arm64`) from the
[latest release](https://github.com/ville6000/toggl-cli/releases/latest),
extract `toggl-cli.exe` and put it in a folder on your `PATH`.

The `gh attestation verify` line (needs the [GitHub CLI](https://cli.github.com))
checks the archive was built by this repo's release workflow; skip it if you
don't have `gh`.

The binaries aren't code-signed. On macOS, a binary downloaded with a browser
rather than `curl` is quarantined by Gatekeeper; allow it with
`xattr -d com.apple.quarantine toggl-cli`.

## With Go

With Go 1.26 or newer:

```sh
go install github.com/ville6000/toggl-cli@latest
```

This installs `toggl-cli` into `$(go env GOPATH)/bin`.

## Shell completion

`toggl-cli completion` prints a completion script for bash, zsh, fish or
PowerShell. For example, for zsh:

```sh
toggl-cli completion zsh > "${fpath[1]}/_toggl-cli"
```

See `toggl-cli completion <shell> --help` for each shell's instructions.
