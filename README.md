# Clewkeep

> Follow the clew back to the evidence.

Clewkeep is a local-first, read-first catalog for AI agent context already present on a computer. Its command-line interface is `ctx`.

```text
agent-native directories
        ↓ scan and read on demand
global ctx catalog
        ↓ explicit selection
custom context repository
        ↓ explicit push (later)
remote repository (later)
```

The first release has one sharp promise:

> One command discovers existing sessions across supported agents. Any locally authorized agent can search the catalog and return to the exact native evidence—without hooks, MCP, a daemon, or a cloud account.

## Status

Clewkeep `0.1.0-rc.1` is a private-beta candidate. The Go core is intentionally small and has no third-party dependencies.

Initial commands:

```text
ctx scan [--full] [--json]
ctx status
ctx list [--provider <name>] [--project <text>] [--json]
ctx search <query> [--limit <n>] [--json]
ctx show <id-or-name> [--json]
ctx name <id-or-name> <name>
ctx snapshot [--name <name>]
ctx diff --since [latest|snapshot]
ctx doctor
ctx version
```

Initial adapters:

- Codex App / CLI rollout JSONL
- Claude Code transcript JSONL

## Permission model

**Local reads are open by default, writes are explicit, external transfer is strict.**

- `scan`, `list`, `search`, `show`, `status`, `diff`, and `doctor` only read agent-native data.
- `name` and `snapshot` write only to ctx-owned local metadata.
- ctx never modifies native agent transcripts.
- push, telemetry, remote sync, and external publishing do not exist in v0.1.

## Build

```powershell
go test ./...
go build -o bin/ctx.exe ./cmd/ctx
```

```bash
go test ./...
go build -o bin/ctx ./cmd/ctx
```

Use `CTX_HOME` to override the global catalog directory. The default is `~/.ctx`.

Use `ctx scan --full` to bypass the incremental cache, reparse every discovered native session, and replace the catalog and scan cache normally.

Build the six private-beta archives with:

```powershell
.\scripts\build-dist.ps1 -Version 0.1.0-rc.1
```

See [`PRIVATE_BETA.md`](PRIVATE_BETA.md) for installation, privacy, and feedback boundaries.

## Agent development

Read [`AGENTS.md`](AGENTS.md), [`VISION.md`](VISION.md), [`TASKS.md`](TASKS.md), and [`agents/README.md`](agents/README.md). That is the entire Agent infrastructure.

## License

Apache-2.0. See [`LICENSE`](LICENSE).
