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
| `discovery/00-problem-definition.md` | Contract | Discovery answers and problem statement. **SIGNED OFF 2026-07-25; Amendment 1 (guarantees-compiler) and Amendment 2 (trust-boundary) recorded same day; RE-SIGNED-OFF 2026-07-26.** |
| `consolidation.md` | Record | Dated log of doc-consolidation passes (28-day cadence; SessionStart hook warns on lapse). |
| `plans/01-week-one.md` | Plan | Week-one artifact: compiler + Postgres (CNPG) + two guarantee triples (mTLS, RPO) → Flux manifests. Revision A APPROVED 2026-07-26; owner decisions 2026-07-26: rpo fails closed (one live triple ships), acceptance claim narrowed. |
| `dogfood.md` | Record | Dated dogfood notes: time-to-stack vs the <1h target (Q2.1), kill-review evidence (Q2.2). First dogfood week opened 2026-07-26; kill review due 2026-08-23. |
| `plans/04-week-four.md` | Plan | The mesh is the substrate: compiled egress (declare + deny enforced, contract over plan-deferral), mesh-mandatory managed scenario, isolated + manual quickstart routes. **DRAFT — awaiting owner approval.** |
| `plans/03-week-three.md` | Plan | Durability triple whole (external store → CONDITIONAL rpo, backup probe passes); healthChecks; quick-start + d7s binary in the dev shell; operator affordances. **APPROVED Revision 1, 2026-07-26.** |
| `plans/02-week-two.md` | Plan | Managed seam proof: `placement: managed` → tofu-controller Terraform CR → Neon free tier, branch-per-stack; Flux-path harness wiring; nixified actions. **APPROVED, at Revision 4 2026-07-26** (revision history and flagged reversals recorded in the plan). |

## Reading order

1. New to the project → `../README.md`, then `foundations/lessons-from-platformctl.md`
   (why the restart), then `discovery/00-problem-definition.md` (where we are).
2. About to build something → stop; check `discovery/00-problem-definition.md` status
   first, then `foundations/golden-rules.md`.
