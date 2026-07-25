---
globs:
  - "docs/**/*.md"
---

# Docs rules (loaded when touching docs/)

- Every doc has exactly one class — contract / plan / record — declared on the docs map
  (`docs/README.md`). A new doc gets its row **in the same commit** that creates it.
- **Records are append-only**: append dated facts, never revise meaning. **Contracts**
  change only as a deliberate act — a dated amendment naming what changed and why, never
  a task side effect. A guard hook enforces this; the human unlock is
  `touch .claude/docs-unlock` (gitignored — delete it when the authorized pass is done).
- Owner answers and scope decisions are recorded **verbatim first, dated**, then
  synthesized. Contradictions are flagged as findings, never reconciled silently.
- Struck questions/rules keep their text with a dated reason — nothing is deleted.
- Scope beyond the signed-off problem definition reopens that contract via dated
  amendment before any work proceeds (golden rule 4).
- Consolidation passes (28-day cadence) are recorded in `docs/consolidation.md`.
