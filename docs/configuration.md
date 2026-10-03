# Configuration

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

## Environment variables

Any setting can also come from an environment variable named `TOGGL_CLI_`
followed by the key in upper case with `.` replaced by `_`. Variables override
the config file, so the CLI can run without one (in CI or a container, say):

```sh
export TOGGL_CLI_TOGGL_TOKEN=<your_api_token>
export TOGGL_CLI_TOGGL_WORKSPACE_ID=1234567
```

Project directory links (`projects`) can only be set in the config file.

## Ticket numbers

When `start` gets no description, it looks for a ticket number in the current
directory's name. By default that is a standalone run of digits: `ticket-123`
and `AB#123` give `123`, while `php8` and `v2` give nothing. A name with more
than one candidate, such as `proj-2024-fix-123`, gives no description and a
warning rather than a wrong guess.

Set `start.ticket_pattern`, or `ticket_pattern` on a project, to a regular
expression for your own format. Its first capture group (or the whole match, if
it has none) becomes the description.

## Project cache

Project lists are cached for 24 hours. When a command can't find a project
name or ID in the cache, it fetches the list again, so new and renamed projects
show up right away. `toggl-cli projects list --refresh` always fetches a fresh
list.
