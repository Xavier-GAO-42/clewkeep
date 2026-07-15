# Agent roles

Keep this small: four workstreams, four branches, one integrator.

| Branch | Suggested model | Job | Output |
| --- | --- | --- | --- |
| `agent/market` | search-strong Gemini-class model | competitors, naming, user pain, primary evidence | concise research note and positioning critique |
| `agent/core` | Codex engineering model | CLI, catalog, search, naming | code and tests |
| `agent/qa` | different strong model family | malformed files, privacy, cross-platform behavior | failing tests and review |
| `agent/launch` | GPT/Claude cross-review | README, GIF script, comparison, launch copy | launch assets with verified claims |

The product father chooses direction and approves anything external. The integrator merges branches. Agents do not vote on product decisions.

Start every model with `AGENTS.md`, this file, and one unchecked item from `TASKS.md`. End with a short message containing branch, files, tests, risks, and next action.
