# aw-watchers

Custom [ActivityWatch](https://activitywatch.net/) watchers.

## Structure

### Watchers

- [aw-watcher-agents](./aw-watcher-agents/) - Logs running coding agents (Claude Code, Codex, OpenCode) via [`abtop --json`](https://github.com/graykode/abtop)

### Utils

- [aw](./internal/aw/) - Minimal ActivityWatch REST API client (hostname, buckets, insert/replace events)

## aw-watcher-agents

Polls `abtop --json` every 15s and writes to the `aw-watcher-agents_<host>` bucket.
One event per session per status (`Executing`, `Thinking`, `Waiting`, `Done`), grown in place while the status holds, with agent, project, cwd, branch, model and session title.
Sleep gaps are not counted.

Flags:
- `--testing` - use the aw-server testing instance (port 5666), passed by aw-qt in testing mode
- `--abtop` - path to `abtop` (default `/opt/homebrew/bin/abtop`, aw-qt starts watchers without the shell PATH)

## Setup

aw-qt discovers `aw-*` executables in its app bundle, PATH, `/opt/homebrew/bin`, `/usr/local/bin` and `~/.local/bin`.
`mise.toml` sets `GOBIN` to `~/.local/bin` (mise's default GOBIN is Go-version-specific and not searched).

```sh
mise run install   # go install ./... into ~/.local/bin
```

Then enable autostart in `~/Library/Application Support/activitywatch/aw-qt/aw-qt.toml`:

```toml
[aw-qt]
autostart_modules = ["aw-server", "aw-watcher-afk", "aw-watcher-window", "aw-watcher-agents"]
```

After reinstalling, restart the watcher from the ActivityWatch tray (Modules) or restart ActivityWatch.

Data shows up in the web UI under **Timeline** (the Activity view only categorizes window/browser data).

## Development

```sh
mise run test      # go test ./...
go vet ./...
go fmt ./...

# run in the foreground against the live server
go run ./aw-watcher-agents
```

## Related

Not in this repo (they need to live elsewhere):
- `~/.zsh/activitywatch.zsh` (dotfiles) - logs every shell command to `aw-watcher-shell_<host>`
- `~/.hammerspoon/modules/aw_away.lua` (dotfiles) - asks what you were doing after AFK periods
