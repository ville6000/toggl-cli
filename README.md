# Toggl CLI

## Configuration

`.toggl-cli` file in the home directory. The file should contain the following:

```yaml
toggl:
  token: <your_api_token>
  workspace_id: <your_workspace_id>
```

The configuration can be generated using the `toggl-cli config` command.
The token can be obtained from the Toggl website.

### 7pace Timetracker (optional)

To post worklogs to an on-prem 7pace Timetracker instance, add a `sevenpace`
section to the config (also prompted for by `toggl-cli config`):

```yaml
sevenpace:
  base_url: https://timetracker.example.com:8090/api/YourCollection/rest
  domain: CORP
  username: <windows_username>
  password: <windows_password>
  activity_type_id: <optional_activity_type_uuid>
  insecure_skip_verify: false # set true only for self-signed corp certs
```

The on-prem 7pace REST API uses NTLM (Windows) authentication, so the domain
username and password are stored in the config file **in plaintext**. Reaching
the instance also requires being on the corporate network / VPN.

## Commands

- `toggl-cli workspaces` - List all workspaces
- `toggl-cli start` - Start a new time entry
- `toggl-cli current` - Show the current time entry
- `toggl-cli continue` - Continue the last time entry
- `toggl-cli stop` - Stop the current time entry
- `toggl-cli history` - List time entries
- `toggl-cli projects` - List projects
- `toggl-cli www` - Open the Toggl website
- `toggl-cli config` - Generate config for the CLI tool
- `toggl-cli 7pace sync` - Post Toggl entries for a date range to 7pace as worklogs
- `toggl-cli 7pace add` - Post a single worklog to 7pace

### 7pace worklogs

`toggl-cli 7pace sync` fetches your Toggl entries and posts each one to 7pace.
The Azure DevOps work item id is parsed from the entry description (e.g.
`#1234 fix bug`, `AB#1234 ...`, or a leading `1234 - ...`); entries without a
work item id are skipped and reported. It accepts the same date flags as
`history` (`--week`, `--month`, `--start`, `--end`).

There is **no de-duplication** — re-running the same range creates duplicate
worklogs. Always preview first with `--dry-run`:

```sh
toggl-cli 7pace sync --week --dry-run   # preview
toggl-cli 7pace sync --week             # post (asks for confirmation)
```

`toggl-cli 7pace add` posts a one-off worklog:

```sh
toggl-cli 7pace add --work-item 1234 --duration 1h30m --comment "code review"
```
