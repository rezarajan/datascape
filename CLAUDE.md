# Datascape (d7s)

Successor to platformctl. **Problem definition SIGNED OFF 2026-07-25, AMENDED same day
(Amendment 1: guarantees-compiler reframing; Amendment 2: trust-boundary model) —
RE-SIGNED-OFF 2026-07-26** (`docs/discovery/00-problem-definition.md` — a contract).
**Golden-rules review (Q3.4) done 2026-07-25**: 67/70 rules bind; see the dated
amendment in `docs/foundations/golden-rules.md`. **Current phase: week-one artifact —
plan at `docs/plans/01-week-one.md`, Revision A APPROVED 2026-07-26. Building now.**

## The one invariant (this phase)

The week-one plan (`docs/plans/01-week-one.md`, Revision A) is APPROVED — product code
is authorized, scoped exactly to Revision A's build order and exit criteria. Scope
beyond the signed-off problem definition, or beyond Revision A's slice, reopens that
contract via dated amendment first.

## Pre-coding checklist

Run before any product change; subagent prompts point here too.

1. Phase + invariant above — is this work authorized in this phase?
2. Scope: inside the signed-off problem definition? Growth re-answers "who asked?" via
   dated amendment first (golden rule 4).
3. Rules: read the golden-rules section for the layer you touch, plus the 2026-07-25
   amendment — struck rules 21/25/26 have reopen criteria (any mutating adapter or
   owned state reinstates them); eight rules carry compiler-shape interpretations.
4. Acceptance: the worked scenario stays runnable exactly as a user invokes it
   (rule 41) — never verified "from memory" (rule 58).
5. Tests: golden-file/conformance coverage for the surface touched (rules 22, 38, 45);
   a live-caught bug lands with a contract-level repro in the same commit (rule 39).
6. Deviations from plan are findings — stop at the smallest consistent state and
   report; never silently adapt scope (agentic-development §5).

## Docs rules

- `docs/README.md` classifies every doc as contract / plan / record. Records are
  append-only. Contracts change only as a deliberate, dated amendment — enforced by a
  guard hook; human unlock is `touch .claude/docs-unlock` (gitignored, logged).
- Discovery answers are recorded inline in `docs/discovery/00-problem-definition.md`,
  dated, verbatim before synthesis. Struck items keep their text with a reason.
- Doc-consolidation passes are recorded in `docs/consolidation.md` (28-day cadence).

## Working in this phase

- The amended problem definition is the scope authority: **d7s is a guarantees
  compiler** — declared guarantees ship as triples (compile-time check + emitted infra +
  conformance probe) or not at all. Zero-trust is the flagship guarantee family, not the
  product identity. Placement (managed vs k8s) is a declared binding; v1 proves exactly
  ONE seam pair (Postgres: CNPG and managed), everything else compiles to the Flux/k8s
  target only. GitOps-compiler posture unchanged (no owned reconcile loop, no mutating
  verbs, no owned state); lakehouse is the acceptance workload; refused:
  arbitrary-substrate abstraction, day-2 operation, TEE, self-serve.
- Trust boundary (Amendment 2): inside = d7s-compiled, by provenance. The outside
  crosses only through named `external` declarations (d7s's `source()` — never
  provisioned or mutated). Egress is compiled default-deny; allowlists come only from
  declared wiring. Security guarantees refuse to compile across the wall; durability/
  freshness may compile as labeled CONDITIONAL with a boundary probe. v1 = declare +
  deny; boundary probes at skeleton; import ceremony v2+.
- New owner answers are recorded verbatim first, then synthesized; contradictions are
  flagged as findings, not reconciled silently.
- Order of work: owner re-signed Amendment 1 + Amendment 2 and approved week-one plan
  Revision A (2026-07-26) → build week-one artifact (compiler + Postgres via CNPG +
  the mTLS guarantee triple, compiled to Flux manifests; `placement: managed` AND
  `guarantees.rpo` both fail closed — owner decisions 2026-07-26, recorded in the
  week-one plan) → week two proves the seam pair's managed side →
  build out the skeleton. Kill review at 4 weeks / 2+ real stacks from the first
  dogfood week.
- Open item from sign-off: the team name/denominator (Q1.1a) — owner records it before
  the first dogfood week.
- Process enforcement lives in `.claude/` (hooks derive their rosters from
  `docs/README.md` and agent frontmatter — never duplicate a list into a hook).
  Multi-session tasks keep `TASK_PROGRESS.md` at the repo root (agentic-development §4).
- Commits follow strict google (conventional) style — `type(scope): subject`,
  lowercase imperative subject, 72-char wrap — enforced by the versioned
  `.githooks/commit-msg` gate (activated per-checkout at session start).

## Compact instructions

When compacting, preserve: owner answers/decisions received this session (verbatim),
the status of `00-problem-definition.md` and of the week-one plan approval, any open
contradiction or design question, and TASK_PROGRESS.md step status. Discard exploratory
file-reading history.
