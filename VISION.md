# Vision and boundary

The project is an open, local context-addressing layer for AI agents.

```text
native Agent directories
        ↓ scan and read on demand
global catalog
        ↓ explicit selection (later)
custom context repo
        ↓ explicit push (later)
remote repo
```

## Constitution

**Local reads are open by default, writes are explicit, external transfer is strict.**

- Local-first, read-first, evidence-first.
- Index native data instead of taking custody of it.
- Keep a small common model and preserve provider-specific facts.
- CLI and versioned JSON are the universal Agent interface.
- No mandatory hook, MCP, daemon, account, or cloud.
- Models may later propose merges; evidence and explicit approval decide.

## v0.1

One command discovers existing Codex and Claude Code sessions. Users and Agents can list, search, inspect, name, snapshot, and structurally diff them without modifying native data.

Not in v0.1: remote sync, semantic merge, native migration, rich GUI, telemetry, complex permissions.

The tool should feel quiet, exact, fast, and durable. It is a gate, not a palace.
