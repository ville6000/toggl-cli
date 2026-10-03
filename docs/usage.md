# Usage

Run `toggl-cli <command> --help` for all of a command's flags.

## Starting timers from a project directory

Link a directory to a project once:

```sh
cd ~/Code/alpha
toggl-cli projects add-path Alpha
```

From then on, `toggl-cli start` inside that directory (or any subdirectory)
uses the Alpha project without `-p`. Without a description, `start` takes the
ticket number from the directory name, so starting a timer in
`~/Code/alpha/ticket-1234` describes it as `1234`. See
[Ticket numbers](configuration.md#ticket-numbers) to match your own ticket format.

## History

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

## Editing and continuing entries

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
