# Clewkeep 五人私测说明

这是 `Clewkeep 0.1.0-rc.2` 的小范围私测候选，不是公开发布。二进制名称为 `ctx`。发行包覆盖 Windows、macOS、Linux，每个平台各有 `amd64` 和 `arm64` 两种架构。

## 安装与卸载

### Windows

1. 解压 `clewkeep-0.1.0-rc.2-windows-<arch>.zip`。
2. 将其中三个文件放在仅自己可访问的目录，例如 `%LOCALAPPDATA%\Clewkeep\`。
3. 直接在该目录运行 `ctx.exe`，或自行将目录加入用户 `PATH`。

卸载：删除该目录，并从用户 `PATH` 中移除它（如果添加过）。

### macOS

1. 解压：`tar -xzf clewkeep-0.1.0-rc.2-darwin-<arch>.tar.gz`。
2. 安装：`mkdir -p ~/.local/bin && install -m 755 ctx ~/.local/bin/ctx`。
3. 确认 `~/.local/bin` 在 `PATH` 中。此私测包未签名；若系统阻止运行，请记录安全警告并停止，不要绕过系统保护。

卸载：`rm ~/.local/bin/ctx`。

### Linux

1. 解压：`tar -xzf clewkeep-0.1.0-rc.2-linux-<arch>.tar.gz`。
2. 安装：`mkdir -p ~/.local/bin && install -m 755 ctx ~/.local/bin/ctx`。
3. 确认 `~/.local/bin` 在 `PATH` 中。

卸载：`rm ~/.local/bin/ctx`。

`<arch>` 选择 `amd64` 或 `arm64`。安装前应按 `SHA256SUMS` 核对下载文件的 SHA-256。

## 从 RC1.1 升级

- 直接运行 `ctx scan`。RC1.1 的 schema 0.1 cache 不会被复用，原生文件会自动重新解析并生成 schema 0.2 catalog。
- 其他命令若读到旧 catalog，会明确要求先运行 `ctx scan`。
- 旧名称只在目标能确定映射到唯一主会话时迁移；目标缺失或有歧义时会拒绝。下一次成功执行 `ctx name` 后，名称索引写为 schema 0.2。
- 旧快照不能参与 RC1.2 diff。先运行 `ctx snapshot` 创建新快照，再使用 `ctx diff`。

RC1.2 的 canonical record ID 形如 `codex/<nativeThreadId>`、`claude/<sessionId>` 或 `claude/<sessionId>/agent/<agentId>`。它与原生路径分离；搜索结果给出的 ID 可以直接用于 `ctx show` 和 `ctx name`。

## 5 分钟首次验证

依次运行：

```text
ctx version
ctx doctor
ctx scan
ctx search <你记得的一段关键词>
ctx search <同一关键词> --provider <codex 或 claude-code>
ctx search <同一关键词> --project <项目路径片段>
ctx show <搜索结果中的 canonical record ID>
ctx scan --full
```

`ctx scan` 和 `ctx status` 显示的是 **indexed records（已索引记录）**，不是独立会话数量；Claude 主会话和 subagent 可以各占一条记录。`--provider` 与 `--project` 同时使用时按 AND 过滤，并在 `--limit` 之前生效。`ctx scan --full` 会跳过增量缓存，重新解析每一个已发现的原生记录文件，并重建 catalog 和 cache。

## 清理本地数据

`CTX_HOME` 是 ctx 自己的本地数据目录。若未设置，默认是用户主目录下的 `.ctx`。

- PowerShell：`Remove-Item -Recurse -Force $env:CTX_HOME`；未设置时删除 `$HOME\.ctx`。
- macOS/Linux：`rm -rf "$CTX_HOME"`；未设置时删除 `~/.ctx`。

执行前请再次确认路径只指向 ctx 的目录。清理会删除 catalog、cache、名称和快照，但不会删除原始 Agent 会话或 transcript。

## 隐私边界 🔒

catalog 和搜索结果都属于敏感本地数据，可能包含项目路径、canonical record ID、原生 session/agent ID 和原文片段。它们应留在测试者自己的电脑上。

不要上传 catalog、cache、原始会话、搜索结果全文或截图。遇到问题时，只报告类别和计数；不要粘贴真实路径、会话内容、身份信息或凭据。

## 五人私测只收集这些信息

- 是否在 5 分钟内安装成功。
- 首次发现多少条已索引记录和多少个项目，只记录两个数字；不要把记录数称为会话数。
- 是否在 60 秒内找到一段以前找不到的会话。
- 搜索结果是否准确回到原始证据。
- 是否出现漏扫、崩溃、权限问题或安全警告。
- 一周后是否愿意再次使用。

未经产品之父明确批准，不联系测试者、不分发文件、不上传数据，也不发布任何内容。
