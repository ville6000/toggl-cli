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
