# GROWTH — 1000-star launch plan

> Internal planning document. All external actions require product-father approval.

## Core insight

The pitch is not "another dev tool" — it's **"the utility that should already exist."** Every developer using Codex and Claude Code has lost an old session. The emotional trigger is recognition, not education.

## Success formula

```
1000 stars = HN front page (300-500)
           + Reddit/community (100-200)
           + X/Twitter viral thread (100-200)
           + Chinese platforms (100-200)
           + sustained content (100-200)
```

This requires hitting at least 3 of these 5 channels effectively.

---

## Phase 0: Pre-launch checklist (1-2 weeks)

Everything below is gated on these being done first.

- [ ] Clean-install test on Windows, macOS, Linux — record and fix every failure
- [ ] 30-second demo GIF: `ctx scan` → `ctx search` → `ctx show` on a real machine with 100+ records
- [ ] 5 private testers complete the full flow; collect 3 usable quotes
- [ ] GitHub repo polished: topics set, description filled, social preview image, Releases page with binaries
- [ ] One-line install script or `go install` path working

### Demo GIF spec

The GIF is the single most important launch asset. It must show:

1. `ctx scan` — "scanned: 347 indexed records, 2 providers, 28 projects" (real numbers create credibility)
2. `ctx search "why we rejected sqlite"` — shows matching result with native path + line
3. `ctx show codex/abc123` — full record with evidence path

Record with [asciinema](https://asciinema.org/) or [vhs](https://github.com/charmbracelet/vhs), convert to GIF. Keep under 15 seconds for the loop. Real data counts, not synthetic.

---

## Phase 1: Launch Day

### Timing

**Tuesday–Thursday, 14:00–15:00 UTC** (6-7am PT, best HN window). Avoid US holidays and major tech announcements.

### Channel 1: Hacker News (primary driver)

Title: `Show HN: Clewkeep – search the AI-agent history already on your machine`

Post:

```text
I kept losing old work between Codex and Claude Code. Each tool can resume
its own sessions, but the histories sit in separate local JSONL stores.
I wanted one small, scriptable way to search across both.

Clewkeep scans existing native session directories on demand — it finds
history that predates installation. `ctx search` returns the original
provider-owned path and line number. It does not rewrite transcripts.

v0.1 is a local CLI: no hook, daemon, MCP, cloud, or telemetry. Zero
third-party Go dependencies.

Spool (https://github.com/spool-lab/spool) is the more complete product
if you want a GUI, four providers, live watching, or secret scanning.
Clewkeep tests whether a smaller explicit-scan CLI is useful enough.

I would value feedback from anyone who searched for an old AI session
in the last two weeks: what were you looking for, and how did you find it?
```

**Key HN tactics:**
- Mention the competitor (Spool) honestly — HN rewards intellectual honesty, punishes omission
- End with a genuine question — drives comments, comments drive ranking
- Do not ask for stars or upvotes anywhere
- Be in the thread replying within 30 minutes of posting; answer every question in detail
- If someone finds a bug or sharp edge, fix it live and post the commit link

### Channel 2: Reddit (day 1, staggered by 2-4 hours after HN)

| Subreddit | Angle |
| --- | --- |
| r/ClaudeAI | "I built a CLI to search across Claude Code sessions and Codex history" |
| r/ChatGPT or r/OpenAI | "CLI to find old Codex sessions — also works with Claude Code" |
| r/commandline | "Local CLI to search AI agent JSONL history across tools" |
| r/golang | "Zero-dependency Go CLI for indexing local AI agent sessions" |

**Each post is written specifically for that community's voice.** r/golang cares about the Go implementation. r/ClaudeAI cares about the Claude Code pain point. Do not cross-post identical text.

### Channel 3: X/Twitter (day 1, morning)

Thread structure:

```
1/ I lost an important AI session last week. It was somewhere between
   Codex and Claude Code on my machine, buried in JSONL files I couldn't
   search.

   So I built Clewkeep. [GIF]

2/ One command scans every Codex and Claude Code session already on
   your machine. No hooks, no daemon — it finds history that predates
   installation.

   ctx scan → 347 records, 28 projects, 2 providers

3/ Search returns the native file path and line number. You go back to
   the original evidence, not a summary or copy.

   ctx search "why we rejected sqlite" → codex/abc123, line 47

4/ It's a Go CLI. Zero dependencies. JSON output for scripts and agents.
   Apache-2.0.

   [link to repo]

5/ Spool by @paperboytm does more (GUI, 4 providers, live watching).
   Clewkeep tests a narrower bet: explicit scan, evidence-first, CLI only.

   Honest comparison in the README.
```

**Tag people who would care** — AI dev tool reviewers, prolific Claude/Codex users. Don't spam; 3-5 targeted mentions max.

### Channel 4: Chinese platforms (day 1-2)

#### V2EX (Creative 或 Python 节点)

标题：`做了个本地 CLI，一条命令搜遍 Codex 和 Claude Code 的历史会话`

```text
最近频繁在 Codex 和 Claude Code 之间切换，发现旧会话很难找回。
两家的历史分散在 ~/.codex/sessions 和 ~/.claude/projects 的 JSONL 里，
原生 resume 只搜自家的、还常限当前项目。

做了 Clewkeep：装上就能索引电脑里已有的全部会话，不是从安装后才开始记录。
搜索结果返回原始文件路径和行号，直接回到证据。

零依赖 Go 二进制，不要 hook、daemon、MCP、云账号，不改写原始会话。

Spool 功能更全（GUI、4 个 Agent、实时监听、泄密扫描），
Clewkeep 验证的是更小的路径：显式扫描、证据优先、纯 CLI。

想听听大家：过去两周你找过旧的 AI 会话吗？花了多久？

GitHub: [link]
```

#### 即刻 (Jike)

简短，带 GIF，#AI工具 #开发者工具 标签。

#### 掘金 (Juejin)

写一篇 800-1200 字的技术短文："跨 Agent 找回旧会话——Clewkeep 的设计思路"。包含架构图、命令示例、和 Spool 的诚实对比。

#### 少数派 (sspai)

适合写"工作流"类文章："AI 编程时代的上下文管理：我为什么做了一个本地优先的会话索引"。

---

## Phase 2: Sustained growth (weeks 2-6)

Launch day peak decays fast. Sustained growth needs recurring content.

### Weekly content rhythm

| Week | Content | Channel |
| --- | --- | --- |
| 2 | "What I learned launching a dev CLI on HN" (post-mortem) | Blog/X/V2EX |
| 3 | Add Gemini CLI adapter → announce as update | X/Reddit/GitHub Discussions |
| 4 | "How Clewkeep works: 500 lines of Go that index JSONL" (technical deep dive) | Blog/Juejin/r/golang |
| 5 | Video demo (2-3 min, real workflow) | YouTube/Bilibili |
| 6 | "Month 1: N users, M sessions indexed, what I'm building next" | X/HN comment/V2EX |

### GitHub Trending

To hit GitHub Trending (Go or overall), you need **sustained daily stars**, not a one-day spike. Tactics:

- Keep the README GIF above the fold — every visitor should see the tool working in < 3 seconds
- Add GitHub Topics: `ai`, `cli`, `codex`, `claude-code`, `developer-tools`, `local-first`, `golang`
- Respond to every issue within 24 hours
- Ship visible improvements weekly — each release is a reason to share again

### Adapter expansion (growth multiplier)

Each new adapter unlocks a new audience:

| Adapter | Audience unlocked | Priority |
| --- | --- | --- |
| Gemini CLI | r/Bard, Gemini users | High — large audience, weak native history |
| Cursor | r/cursor, huge installed base | High — pain point is real |
| Windsurf | Windsurf community | Medium |
| Aider | r/aider, CLI-first users | Medium — natural fit |
| OpenCode | Niche but vocal | Low |

**Announce each adapter as a separate event.** "Clewkeep now indexes Cursor sessions" is a fresh story for an entirely new audience.

### Collaborations

- **Spool**: Propose interop, not competition. If Spool exports, Clewkeep could index. Mutual linking in READMEs.
- **AI tool reviewers on YouTube/Bilibili**: Send binary + 30-second script. Target channels with 5k-50k subscribers (responsive, engaged audiences).
- **Newsletter features**: Console.dev, TLDR newsletter, Changelog, AI weekly newsletters. Submit via their intake forms.

---

## Phase 3: Compounding (weeks 6-12)

### If stars plateau at 300-500

- **Product Hunt launch** — a separate event from HN, typically good for 50-150 stars
- **"Awesome" list PRs** — submit to awesome-cli-apps, awesome-go, awesome-ai-tools
- **Conference lightning talks** — local meetups, virtual Go/AI meetups
- **Integration stories** — "How I use Clewkeep inside Claude Code's CLAUDE.md" tutorial

### If stars are growing past 500

- **GitHub Sponsors** — signals legitimacy, not revenue
- **Discord/community** — only worth the maintenance cost above ~500 active users
- **Contributor guide** — make it easy for people to add adapters (this is the #1 contribution path)

---

## Anti-patterns to avoid

1. **Don't buy stars or use star-bait.** Inflated numbers without real users produce zero retention and damage credibility permanently.
2. **Don't launch before the demo GIF is compelling.** The GIF does 80% of the selling. A wall of text README loses visitors in 5 seconds.
3. **Don't post the same text on every platform.** Each community has its own voice. HN is analytical, Reddit is casual, V2EX is technical-Chinese, 即刻 is punchy.
4. **Don't ignore Spool.** The HN crowd will find it in minutes. Mention it first, honestly. Being the "honest underdog" is a stronger position than pretending the competition doesn't exist.
5. **Don't delay for perfection.** Ship when it works on 3 platforms, has the demo GIF, and 5 real testers have used it. Perfect is the enemy of launched.
6. **Don't neglect Chinese platforms.** V2EX + 即刻 + 掘金 together can deliver 100-200 stars. The Chinese AI dev community is massive, fast-growing, and underserved by English-first tools.

---

## Metrics to track

| Metric | Day 1 target | Week 1 | Month 1 |
| --- | --- | --- | --- |
| GitHub stars | 100-200 | 300-500 | 800-1200 |
| HN points | 100+ | — | — |
| GitHub unique cloners | 50+ | 200+ | 500+ |
| Issues opened | 5+ | 15+ | 30+ |
| Binary downloads | 30+ | 100+ | 300+ |

---

## Decision points for product father

1. **Launch date** — block when pre-launch checklist is complete
2. **Adapter priority** — Gemini CLI or Cursor first? Each unlocks a different audience
3. **MCP endpoint in v0.2?** — would unblock agent-side distribution but contradicts "no MCP" positioning
4. **Chinese-language community investment** — dedicated WeChat group? Bilibili video?
5. **Budget** — any paid promotion (Product Hunt featured, newsletter sponsorship)?
