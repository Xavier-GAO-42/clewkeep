# Clewkeep 五人私测说明

这是 `Clewkeep 0.1.0-rc.1` 的小范围私测，不是公开发布。二进制名称为 `ctx`。

## 安装与卸载

### Windows

1. 解压 `clewkeep-0.1.0-rc.1-windows-<arch>.zip`。
2. 将其中三个文件放在仅自己可访问的目录，例如 `%LOCALAPPDATA%\Clewkeep\`。
3. 直接在该目录运行 `ctx.exe`，或自行将目录加入用户 `PATH`。

卸载：删除该目录，并从用户 `PATH` 中移除它（如果添加过）。

### macOS

1. 解压：`tar -xzf clewkeep-0.1.0-rc.1-darwin-<arch>.tar.gz`。
2. 安装：`mkdir -p ~/.local/bin && install -m 755 ctx ~/.local/bin/ctx`。
3. 确认 `~/.local/bin` 在 `PATH` 中。此私测包未签名；若系统阻止运行，请记录安全警告并停止，不要绕过系统保护。

卸载：`rm ~/.local/bin/ctx`。

### Linux

1. 解压：`tar -xzf clewkeep-0.1.0-rc.1-linux-<arch>.tar.gz`。
2. 安装：`mkdir -p ~/.local/bin && install -m 755 ctx ~/.local/bin/ctx`。
3. 确认 `~/.local/bin` 在 `PATH` 中。

卸载：`rm ~/.local/bin/ctx`。

`<arch>` 选择 `amd64` 或 `arm64`。安装前应按 `SHA256SUMS` 核对下载文件的 SHA-256。

## 5 分钟首次验证

依次运行：

```text
ctx version
ctx doctor
ctx scan
ctx scan --full
ctx search <你记得的一段关键词>
ctx show <搜索结果中的会话 ID>
```

`ctx scan --full` 会跳过增量缓存，强制重新解析每一个已发现的原生会话文件，并按需修复元数据一致但内容已变化的缓存记录；这一行为已通过 QA 验证。按顺序执行即可，无需额外操作。

## 清理本地数据

`CTX_HOME` 是 ctx 自己的本地数据目录。若未设置，默认是用户主目录下的 `.ctx`。

- PowerShell：`Remove-Item -Recurse -Force $env:CTX_HOME`；未设置时删除 `$HOME\.ctx`。
- macOS/Linux：`rm -rf "$CTX_HOME"`；未设置时删除 `~/.ctx`。

执行前请再次确认路径只指向 ctx 的目录。清理会删除 catalog、cache、名称和快照，但不会删除原始 Agent 会话。

## 隐私边界 🔒

catalog 和搜索结果都属于敏感本地数据，可能包含项目路径、会话标识和原文片段。它们应留在测试者自己的电脑上。

不要上传 catalog、cache、原始会话、搜索结果全文或截图。遇到问题时，只报告类别和计数；不要粘贴真实路径、会话内容、身份信息或凭据。

## 五人私测只收集这些信息

- 是否在 5 分钟内安装成功。
- 首次发现多少会话和项目，只记录两个数字。
- 是否在 60 秒内找到一段以前找不到的会话。
- 搜索结果是否准确回到原始证据。
- 是否出现漏扫、崩溃、权限问题或安全警告。
- 一周后是否愿意再次使用。

发行包只通过维护者批准的私测渠道分发；不要上传测试者的本地数据，也不要将本候选包当作公开正式版本传播。
