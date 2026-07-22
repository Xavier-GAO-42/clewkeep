# Clewkeep

> Follow the clew back to the evidence. 🧵

[English](#english) · [简体中文](#简体中文)

---

## English

Clewkeep is a local-first, read-first CLI for finding AI-agent context that is **already on your computer**.

Used both Codex and Claude Code, then lost track of where an old decision lived? Clewkeep scans their existing local session records, makes them searchable, and takes you back to the original provider-owned evidence.

It does not take over your conversations, rewrite transcripts, or require you to start collecting history from the day you install it.

### What it does

- Scans existing Codex and Claude Code session records
- Searches across supported agents from one local catalog
- Narrows results by provider or project path
- Returns an addressable record plus the native evidence path and line
- Lets you name records, create snapshots, and inspect structural changes
- Provides versioned JSON for local scripts and agents

Initial adapters:

- Codex App / CLI rollout JSONL
- Claude Code transcript JSONL

### Quick start

```bash
# Check that local provider roots are discoverable
ctx doctor

# Scan the agent history already on this computer
ctx scan

# Find a phrase you remember
ctx search "cache design"

# Narrow by provider or project
ctx search "cache design" --provider codex
ctx search "cache design" --project my-project

# Return to an indexed record and its native evidence
ctx show codex/<thread-id>
```

Use the returned record ID directly in later commands:

```bash
ctx name codex/<thread-id> "incremental scan design"
ctx snapshot --name before-refactor
ctx diff --since before-refactor
```

### Commands

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

Every record has a provider-qualified ID, such as `codex/<nativeThreadId>`, `claude/<sessionId>`, or `claude/<sessionId>/agent/<agentId>`. An indexed-record count is not necessarily a session count: Claude main sessions and subagents can each be records.

Run `ctx scan --full` when you need to bypass the incremental cache and reparse every discovered native record.

### Why Clewkeep?

| If you need... | Clewkeep's approach |
| --- | --- |
| To resume one provider's current session | Use the provider's native resume flow; Clewkeep is for finding context across supported providers. |
| A visual library, continuous watching, or more adapters | A full session-library product may fit better. Clewkeep deliberately stays a small explicit-scan CLI. |
| To capture only new sessions after installation | Clewkeep can index local history that already existed before installation. |
| Extracted, cloud-synced “memory” | Clewkeep points back to native evidence instead of uploading or rewriting it. |

### Privacy boundary 🔒

**Local reads are open by default. Writes are explicit. External transfer is strict.**

- `scan`, `list`, `search`, `show`, `status`, `diff`, and `doctor` only read agent-native data.
- `name` and `snapshot` write only Clewkeep-owned local metadata.
- Clewkeep never modifies native agent transcripts.
- v0.1 has no cloud sync, telemetry, MCP server, daemon, hook, or automatic upload.

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

Set `CTX_HOME` to choose where Clewkeep stores its own local catalog. The default is `~/.ctx`.

---

## 简体中文

Clewkeep 是一个本地优先、只读优先的命令行工具，用来找回你电脑里**已经存在**的 AI Agent 上下文。✨

如果你在 Codex 和 Claude Code 之间切换过，又忘了“上周那个方案究竟在哪段会话里”，Clewkeep 会扫描本机已有的会话记录，建立本地索引，并把你带回原始的证据文件。

它不会接管会话，不会改写原始记录，也不要求你从安装当天才开始积累历史。

### 能做什么

- 扫描本机已有的 Codex 与 Claude Code 会话记录
- 在支持的 Agent 间统一搜索
- 按 Agent 或项目路径缩小范围
- 返回稳定记录 ID、原始文件路径与行号
- 为重要记录命名、创建快照、查看结构变化
- 为本地脚本和 Agent 提供版本化 JSON 输出

当前支持：

- Codex App / CLI 的 rollout JSONL
- Claude Code 的 transcript JSONL

### 30 秒上手

```bash
# 确认本机可发现的 Agent 数据目录
ctx doctor

# 扫描电脑里已有的 Agent 历史
ctx scan

# 搜索你还记得的一句关键词
ctx search "缓存方案"

# 只搜索某个 Agent 或某个项目
ctx search "缓存方案" --provider codex
ctx search "缓存方案" --project my-project

# 打开搜索结果，回到原始证据
ctx show codex/<thread-id>
```

搜索结果中的记录 ID 可以直接继续使用：

```bash
ctx name codex/<thread-id> "增量扫描设计"
ctx snapshot --name before-refactor
ctx diff --since before-refactor
```

### 它和其他方案的差异

| 你需要什么 | Clewkeep 的取舍 |
| --- | --- |
| 继续某一家 Agent 的当前会话 | 原生 `resume` 更合适；Clewkeep 的价值是跨已支持 Agent 找历史。 |
| GUI、实时监听或更多适配器 | 完整会话库产品可能更合适；Clewkeep 有意保持为小而明确的 CLI。 |
| 只记录安装之后的新会话 | Clewkeep 可以索引安装前就已经存在的本地历史。 |
| 提取、云同步的“记忆” | Clewkeep 以原始证据为中心：不上传，也不改写原始会话。 |

### 隐私边界 🔒

**本地读取默认开放；写入必须显式；外部传输严格禁止。**

- `scan`、`list`、`search`、`show`、`status`、`diff`、`doctor` 只读取 Agent 原生数据。
- `name` 与 `snapshot` 只写入 Clewkeep 自己的本地元数据。
- Clewkeep 不会修改 Codex 或 Claude Code 的原始会话文件。
- v0.1 没有云同步、遥测、MCP 服务、后台 daemon、hook 或自动上传。

### 当前状态

Clewkeep `0.1.0-rc.2` 仍是**私测候选版**，还不是公开生产版本；Go 核心保持零第三方依赖。

最有价值的反馈不是“看起来不错”，而是一次真实的找回：你找什么、是否找对、花了多久、哪一步失败或困惑。🧭

### 从源码构建

```powershell
go test ./...
go build -o bin/ctx.exe ./cmd/ctx
```

```bash
go test ./...
go build -o bin/ctx ./cmd/ctx
```

使用 `CTX_HOME` 可以指定 Clewkeep 保存自身本地索引的位置；默认目录为 `~/.ctx`。

## License

Apache-2.0. See [LICENSE](LICENSE).
