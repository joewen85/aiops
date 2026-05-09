<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **aiops** (5294 symbols, 11110 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/aiops/context` | Codebase overview, check index freshness |
| `gitnexus://repo/aiops/clusters` | All functional areas |
| `gitnexus://repo/aiops/processes` | All execution flows |
| `gitnexus://repo/aiops/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |
| Work in the Handler area (388 symbols) | `.claude/skills/generated/handler/SKILL.md` |
| Work in the Pages area (371 symbols) | `.claude/skills/generated/pages/SKILL.md` |
| Work in the Cloud area (165 symbols) | `.claude/skills/generated/cloud/SKILL.md` |
| Work in the App area (93 symbols) | `.claude/skills/generated/app/SKILL.md` |
| Work in the Api area (45 symbols) | `.claude/skills/generated/api/SKILL.md` |
| Work in the Middlewaresvc area (35 symbols) | `.claude/skills/generated/middlewaresvc/SKILL.md` |
| Work in the Middleware area (26 symbols) | `.claude/skills/generated/middleware/SKILL.md` |
| Work in the Store area (26 symbols) | `.claude/skills/generated/store/SKILL.md` |
| Work in the Docker area (23 symbols) | `.claude/skills/generated/docker/SKILL.md` |
| Work in the Ai area (13 symbols) | `.claude/skills/generated/ai/SKILL.md` |
| Work in the Hooks area (12 symbols) | `.claude/skills/generated/hooks/SKILL.md` |
| Work in the Service area (8 symbols) | `.claude/skills/generated/service/SKILL.md` |
| Work in the Ws area (6 symbols) | `.claude/skills/generated/ws/SKILL.md` |
| Work in the Executor area (5 symbols) | `.claude/skills/generated/executor/SKILL.md` |
| Work in the Config area (5 symbols) | `.claude/skills/generated/config/SKILL.md` |
| Work in the Components area (4 symbols) | `.claude/skills/generated/components/SKILL.md` |
| Work in the Rbac area (4 symbols) | `.claude/skills/generated/rbac/SKILL.md` |
| Work in the Repository area (4 symbols) | `.claude/skills/generated/repository/SKILL.md` |

<!-- gitnexus:end -->
