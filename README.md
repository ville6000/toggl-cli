# toggl-cli

A command line client for [Toggl Track](https://toggl.com/track/): start, stop
and edit timers, and review your history without leaving the terminal.

```console
$ toggl-cli start "code review" -p Alpha
$ toggl-cli current
$ toggl-cli stop
$ toggl-cli history --week
```

- **Timers that know your project.** Link a directory to a Toggl project once,
  and `toggl-cli start` anywhere under it picks that project.
- **Descriptions from the directory name.** Start a timer in
  `~/Code/alpha/ticket-1234` and it's described as `1234`, or match your own
  ticket format with a regular expression.
- **History as JSON.** `history --json` pipes your entries into `jq` or other
  tools.
- **Fix entries afterwards.** Change the start time, description or project of
  a recent entry, or continue an earlier one.

## Installation

```sh
brew install ville6000/tap/toggl-cli
```

Prebuilt binaries for macOS, Linux and Windows, `go install` and shell
completion are covered in [docs/install.md](docs/install.md).

## Getting started

1. Find your API token at the bottom of your
   [Toggl profile page](https://track.toggl.com/profile).
2. Run the interactive setup, which asks for the token, your default workspace
   ID and timezone:

   ```sh
   toggl-cli config
   ```

   `toggl-cli workspaces` lists your workspace IDs once the token is saved.
3. Link your project directory and start tracking:

   ```sh
   cd ~/Code/alpha
   toggl-cli projects add-path Alpha
   toggl-cli start "write release notes"
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

## Documentation

- [Installation](docs/install.md): binaries, `go install`, shell completion
- [Usage](docs/usage.md): project directories, history and JSON output,
  editing and continuing entries
- [Configuration](docs/configuration.md): config file, environment variables,
  ticket number patterns, project cache

## Contributing

Bug reports and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for setting up a development environment,
running the tests and making a release.

## License

[MIT](LICENSE)
