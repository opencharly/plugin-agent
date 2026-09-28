# AGENTS.md — plugin-agent

Standalone plugin repo for the agent control plane (`kind:agent` /
`kind:agent-team` + `command:agent` / `command:tui` / `command:tmux`). The plugin
is a Go module at `candy/plugin-agent/` (module path
`github.com/opencharly/plugin-agent/candy/plugin-agent`); the root `charly.yml`
only declares `discover: candy` so the repo is a project and its candy is
scanned.

Canonical files:

- `candy/plugin-agent/charly.yml` — the `plugin-agent:` candy entity, its
  `plan:` checks, and the embedded `tui-skill:` skill entity.
- `candy/plugin-agent/` — the Go source: `plugin.go`, `control.go`,
  `resolve.go`, `tui.go`, `tmux_compat.go`, `schema/agent.cue`,
  `params/cue_types_gen.go`, `cmd/serve/main.go`.
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-automation:agent` — the full agent control-plane surface (sessions,
  runs, terminal channels, federation, MCP routing). Load before changing
  `charly agent` behaviour.
- `/charly-automation:tmux` — the typed persistent terminal sessions and the
  tmux compatibility facade.
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the unified Provider model, the per-plugin CUE-schema contract. Load
  before touching any provider or schema.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-agent/` — compile the plugin module.
- `go test ./...` in `candy/plugin-agent/` — the plugin's Go tests.
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.

## Modify this repo

- Edit the `plugin-agent:` candy entity, the Go source, and `schema/agent.cue`
  **together** — the schema is the single source for the kinds' `params/`
  structs.
- The `tui-skill:` entity is the projected source for `/charly-automation:tui`;
  keep it in step with any `command:tui` behaviour change.
- Keep the tmux surface a typed facade — never construct remote tmux shell
  strings or access an operator tmux socket.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
