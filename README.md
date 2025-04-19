# Toggl CLI

## Configuration

`.toggl-cli` file in the home directory. The file should contain the following:

```yaml
toggl:
  token: <your_api_token>
  workspace_id: <your_workspace_id>
```

## Commands

- `toggl-cli workspaces` - List all workspaces
- `toggl-cli start` - Start a new time entry
- `toggl-cli current` - Show the current time entry
