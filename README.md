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

> One command discovers existing context as indexed records across supported agents. Any locally authorized agent can search the catalog and return to the exact native evidence—without hooks, MCP, a daemon, or a cloud account.

## Status

Clewkeep `0.1.0-rc.2` is a private-beta candidate, not a public release. The Go core is intentionally small and has no third-party dependencies.

Initial commands:

```text
ctx scan [--full] [--json]
ctx status [--json]
ctx list [--provider <name>] [--project <text>] [--json]
ctx search <query> [--provider <name>] [--project <text>] [--limit <n>] [--json]
ctx show <record-id-or-name> [--json]
ctx name <record-id-or-name> <name>
ctx snapshot [--name <name>] [--json]
ctx diff --since [latest|snapshot] [--json]
ctx doctor [--json]
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

## Record identity and RC1.2 upgrade

Every indexed record has a provider-qualified canonical ID: `codex/<nativeThreadId>`, `claude/<sessionId>`, or `claude/<sessionId>/agent/<agentId>`. Search results return this ID, and the same value can be passed directly to `ctx show` or `ctx name`. Schema 0.2 JSON also preserves `native_session_id`, optional `native_agent_id`, and `record_kind` (`session` or `subagent`) as separate native facts.

When upgrading from RC1.1:

- An ordinary `ctx scan` ignores schema 0.1 cache entries and reparses the native files into a schema 0.2 catalog.
- Commands that encounter a schema 0.1 catalog ask you to run `ctx scan`.
- Schema 0.1 names migrate only when their old target maps deterministically to one main session; ambiguous or missing targets are rejected. The next successful `ctx name` write persists schema 0.2 names.
- Schema 0.1 snapshots cannot be diffed. Create a new snapshot before using `ctx diff`.

Use `ctx scan --full` to bypass the incremental cache, reparse every discovered native record, and replace the catalog and scan cache normally.

Build the six private-beta archives with:

```powershell
.\scripts\build-dist.ps1 -Version 0.1.0-rc.2
```

See [`PRIVATE_BETA.md`](PRIVATE_BETA.md) for installation, migration, privacy, and feedback boundaries.

## Agent development

Read [`AGENTS.md`](AGENTS.md), [`VISION.md`](VISION.md), [`TASKS.md`](TASKS.md), and [`agents/README.md`](agents/README.md). That is the entire Agent infrastructure.

## License

Apache-2.0. See [`LICENSE`](LICENSE).
