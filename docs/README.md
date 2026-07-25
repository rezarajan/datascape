# Docs map

Every document in this repo is one of three things. The classification tells you whether a
file may be edited, drawn from, or only appended to. (Rule carried from platformctl, where
mechanically enforcing this distinction kept plans and history honest.)

- **Contract** — what work is checked against. Changing a contract is a deliberate,
  reviewed act, never a side effect of a task.
- **Plan** — where work is drawn from. Plans evolve additively; superseded intent is
  marked, not erased.
- **Record** — append-only history. Append facts, never revise meaning.

| Document | Class | Purpose |
|---|---|---|
| `foundations/golden-rules.md` | Contract | The engineering rules datascape is built under. |
| `foundations/lessons-from-platformctl.md` | Record | Post-mortem of the predecessor project. |
| `foundations/agentic-development.md` | Contract | Development-process rules for agentic work in this repo. |
| `discovery/00-problem-definition.md` | Contract | Discovery answers and problem statement. **SIGNED OFF 2026-07-25; Amendment 1 (guarantees-compiler) recorded same day, awaiting owner re-sign-off.** |
| `consolidation.md` | Record | Dated log of doc-consolidation passes (28-day cadence; SessionStart hook warns on lapse). |
| `plans/01-week-one.md` | Plan | Week-one artifact: compiler + Postgres (CNPG) + two guarantee triples (mTLS, RPO) → Flux manifests. Revision A awaiting owner approval. |

## Reading order

1. New to the project → `../README.md`, then `foundations/lessons-from-platformctl.md`
   (why the restart), then `discovery/00-problem-definition.md` (where we are).
2. About to build something → stop; check `discovery/00-problem-definition.md` status
   first, then `foundations/golden-rules.md`.
