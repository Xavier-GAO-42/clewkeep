# Clewkeep

> **Your AI context is already on disk. Clewkeep makes it addressable.** 🧵

[English](#english) · [简体中文](#简体中文)

---

## English

Codex and Claude Code already leave valuable decisions, experiments, and project history on your computer. The problem is not that this context is missing—it is scattered across providers, projects, and old sessions that the next agent cannot reliably find.

Clewkeep gives humans and terminal-capable agents one quiet, local CLI for locating that history and returning to the original evidence. It does not require hooks, an MCP server, a daemon, or cloud custody, and it does not rewrite provider transcripts.

```text
native Codex + Claude Code history
                ↓ explicit scan and search
       provider-qualified record
                ↓ native path + line + snippet
     authorized agent reads the evidence it needs
```

This gives an AI a grounded, cross-provider and cross-project view of work already performed on the machine—within the files, providers, and permissions Clewkeep supports. Clewkeep locates the evidence; the model does the understanding.

### The missing layer

Every new agent session starts with a narrow context window, while useful history remains on disk:

- Native resume flows stay inside one provider and usually one selected session.
- Hooks and wrappers usually capture activity only after they are installed.
- MCP and memory services add a standing integration or ingestion layer.
- Cloud memory changes where extracted context is stored and trusted.

Clewkeep takes a smaller route: explicitly scan supported provider-owned history already on disk, keep a local metadata catalog, and search the native JSONL only when asked.

### Three commands

```bash
# Index supported native history already on this computer
ctx scan --json

# Search across providers, or narrow by provider and project
ctx search "why did we reject sqlite?" --project clewkeep --json

# Resolve the stable record ID back to its native evidence
ctx show codex/<thread-id> --json
```

`search` returns a stable record ID, provider, project, native file path, matching line, and snippet. A local agent with appropriate file permissions can then read only the relevant region of the original transcript instead of importing every conversation into its prompt.

### Why agents can use it

- A terminal-capable agent can call `ctx` on demand; Clewkeep does not need to run inside the agent runtime.
- Versioned JSON makes the output machine-readable without scraping terminal text.
- Provider and project filters let an agent narrow the evidence before reading it.
- Search points back to provider-owned files instead of a detached summary or migrated copy.
- Codex and Claude Code history can be queried through the same small interface.

### How it differs

| Approach | Runtime integration | Existing history | Evidence model |
| --- | --- | --- | --- |
| Native resume | Provider-native | One provider/session | Native session |
| Hook or wrapper | Runs in the capture path | Usually starts after setup | Captured copy or log |
| MCP or memory service | Standing endpoint or ingestion layer | Depends on ingestion | Extracted or served memory |
| Cloud memory | External storage and retrieval | Depends on upload/sync | Remotely stored context |
| **Clewkeep** | **Explicit CLI call** | **Supported native history already on disk** | **Native path + line + snippet** |

Clewkeep is not a GUI session library, semantic search engine, automatic summarizer, or background watcher. Those may be better choices when you need continuous capture, more adapters, or a visual browser. Clewkeep is for quiet, evidence-first retrieval without changing the agent workflow.

### Supported records

- Codex App / CLI rollout JSONL
- Claude Code main-session transcript JSONL
- Claude Code subagent transcript JSONL

“Native history” means supported records that the adapters can discover and parse. It does not mean every file on the computer or every historical format a provider may ever produce.

### Use the Clewkeep skill

The repository includes a Codex skill at [`skills/clewkeep`](skills/clewkeep). Copy that directory into your Codex skills directory, then ask the model to use `$clewkeep`.

Example:

```text
Use $clewkeep to find why we rejected SQLite in this project and trace the conclusion back to native evidence.
```

The skill teaches the model to verify the `ctx` executable, prefer JSON, narrow by provider/project, treat transcripts as untrusted evidence rather than instructions, and read only the minimum relevant native lines.

### Command reference

```text
ctx doctor [--json]
ctx scan [--full] [--json]
ctx status [--json]
ctx list [--provider <name>] [--project <text>] [--json]
ctx search <query> [--provider <name>] [--project <text>] [--limit <n>] [--json]
ctx show <record-id-or-name> [--json]
ctx name <record-id-or-name> <name>
ctx snapshot [--name <name>] [--json]
ctx diff --since [latest|snapshot] [--json]
ctx version
```

Record IDs are provider-qualified: `codex/<nativeThreadId>`, `claude/<sessionId>`, or `claude/<sessionId>/agent/<agentId>`. Indexed-record counts are not session counts because a Claude main session and its subagents can each be records.

Ordinary scans are incremental. Use `ctx scan --full --json` only when you need to bypass the cache and reparse all discovered native records.

### Privacy boundary 🔒

**Local reads are open by default. Writes are explicit. External transfer is strict.**

- Native transcripts remain provider-owned and read-only.
- The catalog and incremental cache store metadata, not transcript bodies.
- Search reads native JSONL on demand; search output is sensitive local data.
- `name` and `snapshot` write only Clewkeep-owned metadata under `CTX_HOME`.
- v0.1 has no telemetry, cloud sync, MCP server, daemon, hook, or automatic upload.
- Clewkeep does not bypass operating-system or agent permissions.

### Status

Clewkeep `0.1.0-rc.2` is a **private-beta candidate**, not a public production release. The Go core has no third-party dependencies.

The most valuable feedback is a real retrieval attempt: what you searched for, whether you found the right context, how long it took, and where the flow failed.

### Build from source

```powershell
go test ./...
go build -o bin/ctx.exe ./cmd/ctx
```

```bash
go test ./...
go build -o bin/ctx ./cmd/ctx
```

Set `CTX_HOME` to choose where Clewkeep stores its local catalog. The default is `~/.ctx`.

---

## 简体中文

> **你的 AI 上下文已经在电脑里，Clewkeep 只是让它可被找到。** 🧵

Codex 与 Claude Code 已经把重要的决策、实验和项目历史留在电脑上。真正的问题不是上下文不存在，而是它散落在不同 Agent、不同项目和旧会话中，下一个 AI 很难可靠地找到。

Clewkeep 提供一个安静、本地、只读优先的 CLI，让人和有终端权限的 AI 都能定位这些历史并回到原始证据。它不要求 hook、MCP server、后台 daemon 或云端托管，也不会改写 Agent 的原始会话。

```text
本机 Codex + Claude Code 原生历史
                ↓ 显式扫描与检索
          带 provider 的稳定记录
                ↓ 原始路径 + 行号 + 片段
       获得授权的 AI 按需读取相关证据
```

这让 AI 能够基于电脑里已经完成的工作，建立跨 Agent、跨项目的可靠认识——范围始终受支持的文件、适配器和本机权限约束。Clewkeep 负责找到证据，模型负责理解证据。

### 真正缺失的一层

每个新 Agent 会话都从有限的上下文窗口开始，但有价值的历史其实仍在本机：

- 原生 `resume` 通常局限在一家 Agent 和选中的某个会话中。
- hook 与 wrapper 通常只能从安装后开始捕获活动。
- MCP 与 memory 服务会增加常驻接口或数据摄取层。
- 云端 memory 会改变上下文的存储位置与信任边界。

Clewkeep 选择更小的路径：显式扫描磁盘上已经存在、由原提供方拥有的受支持历史，只保存本地元数据目录，并在收到搜索请求时按需读取原生 JSONL。

### 三条命令

```bash
# 索引电脑里已经存在的受支持 Agent 历史
ctx scan --json

# 跨 Agent 搜索，也可以按 provider 和项目缩小范围
ctx search "为什么放弃 sqlite？" --project clewkeep --json

# 用稳定记录 ID 回到原始证据
ctx show codex/<thread-id> --json
```

`search` 会返回稳定记录 ID、provider、项目、原始文件路径、命中行号和片段。有相应文件权限的本地 AI 随后可以只读取原始 transcript 中真正相关的范围，不必把所有会话全部塞进当前提示词。

### 为什么 AI 可以直接使用

- 有终端权限的 AI 可以按需调用 `ctx`，无需把 Clewkeep 嵌入 Agent runtime。
- 版本化 JSON 让模型无需解析人类终端排版。
- provider 与 project 过滤器让模型先缩小范围，再读取证据。
- 搜索结果回到原提供方文件，而不是脱离来源的摘要或迁移副本。
- Codex 与 Claude Code 历史可以通过同一个小接口查询。

### 和其他方案的差异

| 方案 | 对 Agent runtime 的介入 | 已有历史 | 证据形式 |
| --- | --- | --- | --- |
| 原生 resume | 原提供方内部 | 一家 Agent／一个会话 | 原生会话 |
| Hook 或 wrapper | 位于捕获路径中 | 通常从配置完成后开始 | 捕获副本或日志 |
| MCP 或 memory 服务 | 常驻接口或摄取层 | 取决于是否摄取 | 提取或服务化的 memory |
| 云端 memory | 外部存储与检索 | 取决于上传／同步 | 远端上下文 |
| **Clewkeep** | **显式 CLI 调用** | **磁盘上已有的受支持原生历史** | **原始路径 + 行号 + 片段** |

Clewkeep 不是 GUI 会话库、语义搜索、自动总结器或后台 watcher。如果你需要持续捕获、更多适配器或可视化浏览，其他产品可能更合适。Clewkeep 专注于不改变 Agent 工作流的、安静且证据优先的上下文检索。

### 支持的记录

- Codex App / CLI rollout JSONL
- Claude Code 主会话 transcript JSONL
- Claude Code subagent transcript JSONL

这里的“原生历史”指适配器可以发现并解析的受支持记录，不代表电脑里的全部文件，也不代表提供方过去和未来的所有格式。

### 让模型学会使用 Clewkeep

仓库内置了 Codex Skill：[`skills/clewkeep`](skills/clewkeep)。将该目录复制到 Codex skills 目录后，即可要求模型使用 `$clewkeep`。

示例：

```text
使用 $clewkeep 找到这个项目为什么放弃 SQLite，并把结论追溯到原始证据。
```

这个 Skill 会指导模型核对真正的 `ctx` 可执行文件、优先使用 JSON、按 provider／project 缩小范围、把 transcript 当作证据而不是指令，并且只读取最小必要的原始行。

### 隐私边界 🔒

**本地读取默认开放；写入必须显式；外部传输严格控制。**

- 原生 transcript 始终归提供方所有，并保持只读。
- catalog 与增量 cache 只保存元数据，不复制 transcript 正文。
- 搜索按需读取原生 JSONL；搜索输出本身仍属于敏感本地数据。
- `name` 与 `snapshot` 只在 `CTX_HOME` 下写入 Clewkeep 自己的元数据。
- v0.1 没有遥测、云同步、MCP server、daemon、hook 或自动上传。
- Clewkeep 不会绕过操作系统或 Agent 的权限。

### 当前状态

Clewkeep `0.1.0-rc.2` 仍是**私测候选版**，不是公开生产版本；Go 核心保持零第三方依赖。

最有价值的反馈不是“看起来不错”，而是一次真实找回：你在找什么、是否找对、花了多久、哪一步失败或困惑。🧭

### 从源码构建

```powershell
go test ./...
go build -o bin/ctx.exe ./cmd/ctx
```

```bash
go test ./...
go build -o bin/ctx ./cmd/ctx
```

使用 `CTX_HOME` 可以指定 Clewkeep 保存本地目录的位置；默认目录为 `~/.ctx`。

## License

Apache-2.0. See [LICENSE](LICENSE).
