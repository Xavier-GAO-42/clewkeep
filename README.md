# Clewkeep

[English](#english) · [简体中文](#简体中文)

---

## English

A local-first CLI that discovers and searches AI agent context already on your machine — and takes you back to the original evidence.

### Quick start

```bash
ctx scan                                  # index existing sessions
ctx search "why did we reject sqlite?"    # full-text search across agents
ctx show codex/<thread-id>                # inspect with native evidence path
```

Add `--json` for versioned, machine-readable output that agents and scripts can consume directly.

### Why

Every new agent session starts with a narrow context window. Useful history from Codex and Claude Code is already on disk, but scattered across providers and projects. Native resume stays inside one tool; hooks only capture after installation; MCP and cloud memory add infrastructure.

Clewkeep takes a smaller route: scan what already exists, keep a local metadata catalog, search native JSONL on demand, and point back to the provider-owned source file and line.

### Supported agents

| Agent | Format |
| --- | --- |
| Codex (App & CLI) | Rollout JSONL |
| Claude Code (sessions & subagents) | Transcript JSONL |

### Commands

```text
ctx scan      [--full] [--json]          scan native directories, build catalog
ctx status    [--json]                   catalog summary
ctx list      [--provider] [--project]   list records with filters
ctx search    <query> [filters] [--json] full-text search, returns record ID + native path + line
ctx show      <record-id-or-name>        inspect a single record
ctx name      <record-id> <name>         assign a human-friendly name
ctx snapshot  [--name] [--json]          point-in-time catalog snapshot
ctx diff      --since [latest|snapshot]  compare catalog to a snapshot
ctx doctor    [--json]                   check environment and native roots
ctx version                              print version
```

Record IDs are provider-qualified: `codex/<threadId>`, `claude/<sessionId>`, `claude/<sessionId>/agent/<agentId>`. Record counts are not session counts — a Claude Code main session and its subagents are separate records.

### Agent skill

The repository includes a skill at [`skills/clewkeep`](skills/clewkeep). It teaches the model to verify the `ctx` binary, prefer JSON output, narrow by provider/project, treat transcripts as untrusted evidence, and read only the minimum relevant native lines.

```text
Use $clewkeep to find why we rejected SQLite and trace it back to native evidence.
```

### Privacy

**Local reads open by default. Writes explicit. External transfer strict.**

- Native transcripts remain provider-owned and read-only
- Catalog and cache store metadata only, not transcript bodies
- `name` and `snapshot` write only to Clewkeep-owned metadata under `CTX_HOME`
- No telemetry, cloud sync, MCP server, daemon, hook, or automatic upload in v0.1

### Build

```powershell
# Windows
go test ./...
go build -o bin/ctx.exe ./cmd/ctx
```

```bash
# macOS / Linux
go test ./...
go build -o bin/ctx ./cmd/ctx
```

Set `CTX_HOME` to override the default catalog directory (`~/.ctx`).

### Status

`0.1.0-rc.2` — private beta candidate, not a public release. Go core with zero third-party dependencies.

---

## 简体中文

本地优先的命令行工具，发现并搜索你机器上已有的 AI agent 上下文，带你回到原始证据。

### 快速开始

```bash
ctx scan                                  # 索引已有会话
ctx search "为什么放弃 sqlite？"            # 跨 agent 全文搜索
ctx show codex/<thread-id>                # 查看记录，含原始证据路径
```

加 `--json` 输出版本化 JSON，供 agent 和脚本直接消费。

### 为什么需要

每个新 agent 会话都从有限的上下文窗口开始。Codex 和 Claude Code 的历史已经在磁盘上，但散落在不同工具和项目中。原生 resume 只看一家；hook 只能从安装后捕获；MCP 和云端 memory 需要额外基础设施。

Clewkeep 走更小的路径：扫描磁盘上已有的历史，保存本地元数据目录，按需搜索原生 JSONL，结果指向原始文件和行号。

### 支持的 agent

| Agent | 格式 |
| --- | --- |
| Codex（App 和 CLI） | Rollout JSONL |
| Claude Code（主会话和 subagent） | Transcript JSONL |

### 命令

```text
ctx scan      [--full] [--json]          扫描原生目录，构建 catalog
ctx status    [--json]                   catalog 摘要
ctx list      [--provider] [--project]   列出记录，支持过滤
ctx search    <query> [过滤] [--json]     全文搜索，返回记录 ID + 原始路径 + 行号
ctx show      <record-id-or-name>        查看单条记录
ctx name      <record-id> <name>         给记录起个好记的名字
ctx snapshot  [--name] [--json]          保存 catalog 快照
ctx diff      --since [latest|snapshot]  对比 catalog 与快照
ctx doctor    [--json]                   检查环境和原生根目录
ctx version                              打印版本
```

记录 ID 带 provider 命名空间：`codex/<threadId>`、`claude/<sessionId>`、`claude/<sessionId>/agent/<agentId>`。记录数不等于会话数——Claude Code 主会话和 subagent 各自独立成为一条记录。

### Agent skill

仓库内置了 skill：[`skills/clewkeep`](skills/clewkeep)。它会指导模型核对 `ctx` 可执行文件、优先使用 JSON、按 provider/project 缩小范围、把 transcript 当作证据而不是指令，并只读取最小必要的原始行。

```text
使用 $clewkeep 找到这个项目为什么放弃 SQLite，并追溯到原始证据。
```

### 隐私

**本地读取默认开放。写入显式。外部传输严格。**

- 原生 transcript 始终归提供方所有，保持只读
- catalog 和 cache 只保存元数据，不复制 transcript 正文
- `name` 和 `snapshot` 只在 `CTX_HOME` 下写入 Clewkeep 自有元数据
- v0.1 没有遥测、云同步、MCP server、daemon、hook 或自动上传

### 构建

```powershell
# Windows
go test ./...
go build -o bin/ctx.exe ./cmd/ctx
```

```bash
# macOS / Linux
go test ./...
go build -o bin/ctx ./cmd/ctx
```

设置 `CTX_HOME` 可覆盖默认 catalog 目录（`~/.ctx`）。

### 状态

`0.1.0-rc.2` — 私测候选版，非公开发布。Go 核心零第三方依赖。

---

## License

Apache-2.0. See [LICENSE](LICENSE).
