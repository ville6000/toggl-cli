# toggl-cli

A command line client for [Toggl Track](https://toggl.com/track/): start, stop
and edit timers, and review your history.

```console
$ toggl-cli start "code review" -p Alpha
$ toggl-cli current
$ toggl-cli stop
$ toggl-cli history --week
```

- [Installation](#installation)
- [Getting started](#getting-started)
- [Usage](#usage)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Homebrew

On macOS and Linux:

```sh
brew install ville6000/tap/toggl-cli
```

### Prebuilt binaries

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
tar -xzf "toggl-cli_$PLATFORM.tar.gz" toggl-cli
sudo mv toggl-cli /usr/local/bin/
```

On Windows, download `toggl-cli_windows_amd64.zip` (or `_arm64`) from the
[latest release](https://github.com/ville6000/toggl-cli/releases/latest),
extract `toggl-cli.exe` and put it in a folder on your `PATH`.

The binaries aren't signed. On macOS, a binary downloaded with a browser rather
than `curl` is quarantined by Gatekeeper; allow it with
`xattr -d com.apple.quarantine toggl-cli`.

### With Go

With Go 1.26 or newer:

```sh
go install github.com/ville6000/toggl-cli@latest
```

This installs `toggl-cli` into `$(go env GOPATH)/bin`.

### Shell completion

`toggl-cli completion` prints a completion script for bash, zsh, fish or
PowerShell. For example, for zsh:

```sh
toggl-cli completion zsh > "${fpath[1]}/_toggl-cli"
```

See `toggl-cli completion <shell> --help` for each shell's instructions.

## Getting started

1. Find your API token at the bottom of your
   [Toggl profile page](https://track.toggl.com/profile).
2. Run the interactive setup, which asks for the token, your default workspace
   ID and timezone:

   ```sh
   toggl-cli config
   ```

   `toggl-cli workspaces` lists your workspace IDs once the token is saved.
3. Start tracking:

   ```sh
   toggl-cli start "write release notes" -p Docs
   ```

## Usage

| Command | Description |
| --- | --- |
| `toggl-cli start [description]` | Start a timer (`-p` project) |
| `toggl-cli current` | Show the running timer |
| `toggl-cli stop` | Stop the running timer |
| `toggl-cli continue` | Start a new timer like a recent entry |
| `toggl-cli edit` | Change a recent or running entry |
| `toggl-cli history` | Show time logged per day |
| `toggl-cli projects list` | List the workspace's projects |
| `toggl-cli projects add-path <project>` | Link the current directory to a project |
| `toggl-cli workspaces` | List your workspaces |
| `toggl-cli www` | Open Toggl in the browser |
| `toggl-cli config` | Create or update the config file |

Run `toggl-cli <command> --help` for all flags, and `toggl-cli --version` for
the installed version.

### Starting timers from a project directory

Link a directory to a project once:

```sh
cd ~/Code/alpha
toggl-cli projects add-path Alpha
```

From then on, `toggl-cli start` inside that directory (or any subdirectory)
uses the Alpha project without `-p`. Without a description, `start` takes the
ticket number from the directory name, so starting a timer in
`~/Code/alpha/ticket-1234` describes it as `1234`. See
[Ticket numbers](#ticket-numbers) to match your own ticket format.

### History

`toggl-cli history` shows today's time, summed by description and project.
Choose the range with `--day`/`-d`, `--week`, `--month`, or `--start` / `--end`
(`YYYY-MM-DD`, both inclusive). `--verbose` also lists each entry with its ID.

```sh
toggl-cli history --week
toggl-cli history --day 2024-06-03
toggl-cli history --start 2024-06-01 --end 2024-06-07 --verbose
```

`--json` prints the range's individual entries as a JSON array instead, oldest
first, for piping into other tools:

```json
[
  {
    "id": 4123456789,
    "start": "2024-06-03T09:15:00+03:00",
    "duration": 1800,
    "running": false,
    "description": "#1234 code review",
    "project": "Alpha",
    "tags": []
  }
]
```

`start` is in your configured timezone and `duration` is in seconds; a running
entry has `"running": true` and the time elapsed so far. An empty range gives
`[]`.

```sh
toggl-cli history --week --json | jq '[.[] | .duration] | add'
```

### Editing and continuing entries

`toggl-cli edit` changes the description (`-d`), project (`-p`) or start time
(`-s`) of your most recent entry. `toggl-cli continue` starts a new timer with a
recent entry's description and project. Both act on the most recent entry
unless you pick another one:

- `--id <ID>`: the entry's ID, as shown in the ID column of `current`, `stop`
  and `history --verbose`. IDs don't change, and they also reach older
  entries.
- `--index`/`-i <n>`: the position among your recent entries, `0` being the
  most recent. Positions shift whenever a new entry starts.

The start time is read in your configured timezone, as `"YYYY-MM-DD HH:MM"`,
`HH:MM` (keeping the entry's date) or `YYYY-MM-DD`. For a stopped entry the end
time stays fixed and the duration is recomputed.

```sh
toggl-cli edit --start 09:00                     # most recent entry
toggl-cli edit -i 1 --start "2024-06-01 08:30"   # the one before it
toggl-cli edit --id 4123456789 -d "code review"  # any entry, by ID
toggl-cli continue --id 4123456789
```

## Configuration

`toggl-cli` reads its settings from the first of these files that exists:

1. `$XDG_CONFIG_HOME/toggl-cli/config.yaml`
2. `~/.config/toggl-cli/config.yaml`
3. `~/.toggl-cli.yaml`

`toggl-cli config` updates that file, or creates
`~/.config/toggl-cli/config.yaml` (under `$XDG_CONFIG_HOME` when that is set),
readable only by you. Other commands accept `--config <file>` to read a
different file.

```yaml
toggl:
  token: <your_api_token>        # required
  workspace_id: 1234567          # required: default workspace
  timezone: Europe/Helsinki      # optional, defaults to the system timezone

start:
  ticket_pattern: "([A-Z]+-[0-9]+)"  # optional, see "Ticket numbers"

projects:                        # written by `projects add-path`
  Alpha:
    paths:
      - /Users/me/Code/alpha
    ticket_pattern: "task-([0-9]+)"  # optional, overrides start.ticket_pattern
```

### Environment variables

Any setting can also come from an environment variable named `TOGGL_CLI_`
followed by the key in upper case with `.` replaced by `_`. Variables override
the config file, so the CLI can run without one (in CI or a container, say):

```sh
export TOGGL_CLI_TOGGL_TOKEN=<your_api_token>
export TOGGL_CLI_TOGGL_WORKSPACE_ID=1234567
```

Project directory links (`projects`) can only be set in the config file.

### Ticket numbers

When `start` gets no description, it looks for a ticket number in the current
directory's name. By default that is a standalone run of digits: `ticket-123`
and `AB#123` give `123`, while `php8` and `v2` give nothing. A name with more
than one candidate, such as `proj-2024-fix-123`, gives no description and a
warning rather than a wrong guess.

Set `start.ticket_pattern`, or `ticket_pattern` on a project, to a regular
expression for your own format. Its first capture group (or the whole match, if
it has none) becomes the description.

### Project cache

Project lists are cached for 24 hours. When a command can't find a project
name or ID in the cache, it fetches the list again, so new and renamed projects
show up right away. `toggl-cli projects list --refresh` always fetches a fresh
list.

## Contributing

Bug reports and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for setting up a development environment,
running the tests and making a release.

## License

[MIT](LICENSE)
