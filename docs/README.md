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
| `discovery/00-problem-definition.md` | Contract | Discovery answers and problem statement. **SIGNED OFF 2026-07-25** — scope changes reopen it via dated amendment. |

## Reading order

1. New to the project → `../README.md`, then `foundations/lessons-from-platformctl.md`
   (why the restart), then `discovery/00-problem-definition.md` (where we are).
2. About to build something → stop; check `discovery/00-problem-definition.md` status
   first, then `foundations/golden-rules.md`.
