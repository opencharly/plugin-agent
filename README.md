# plugin-agent

The headless, daemon-free agent control plane for OpenCharly — agent and
agent-team kinds plus the reflected `charly agent`, `charly tui`, and
compatibility `charly tmux` commands.

The plugin persists sessions, runs, ordered evidence, incidents, RCA, and
recovery decisions, while routing every runtime and terminal operation through
generic `Provider.Channel` target data. It is compiled into charly
(`command:agent` / `command:tui` / `command:tmux` + `kind:agent` /
`kind:agent-team`). The tmux compatibility surface is a typed facade — it never
constructs remote tmux shell strings or touches an operator tmux socket.

## What it provides

| Capability | Surface |
|---|---|
| `kind:agent` | the `agent:` kind entity |
| `kind:agent-team` | the `agent-team:` kind entity |
| `command:agent` | `charly agent` — sessions, runs, terminal channels, federation, MCP routing, incidents/RCA/recovery |
| `command:tui` | `charly tui` — the interactive terminal UI for the control plane |
| `command:tmux` | `charly tmux` — the typed compatibility facade over persistent terminal sessions |

## How to use it

The commands are compiled in — no candy composition is needed to use them:

```bash
charly agent runtime list
charly tui
charly tmux list
```

For a scripting or single operation use `charly agent`; use `charly tui` for
interactive inspection.

## Layout

- `candy/plugin-agent/` — the plugin module: `plugin.go`, `control.go`,
  `resolve.go`, `tui.go`, `tmux_compat.go`, `schema/agent.cue`,
  `params/cue_types_gen.go`, `cmd/serve/main.go`.
- `candy/plugin-agent/charly.yml` — the `plugin-agent:` candy entity, its
  `plan:` checks, and the embedded `tui-skill:` skill entity.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skills: `/charly-automation:agent` (the full control-plane surface) and
  `/charly-automation:tui` (the `charly tui` UI, projected from the embedded
  `tui-skill:` entity).
- `/charly-automation:tmux` — the typed persistent terminal sessions.
- `/charly-internals:plugin` — the plugin/provider model.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
