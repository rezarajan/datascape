# Datascape (d7s)

Successor to platformctl. **Problem definition SIGNED OFF 2026-07-25**
(`docs/discovery/00-problem-definition.md` — a contract). **Golden-rules review (Q3.4)
done 2026-07-25**: 67/70 rules bind; see the dated amendment in
`docs/foundations/golden-rules.md`. **Current phase: week-one artifact — plan at
`docs/plans/01-week-one.md`, awaiting owner approval. No product code exists yet.**

## The one invariant (this phase)

Product code is written only after the owner approves the week-one plan
(`docs/plans/01-week-one.md`). If a task asks for product functionality before then,
stop. Scope beyond the signed-off problem definition reopens that contract via dated
amendment first.

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

- The signed-off problem definition is the scope authority: one k8s target, Flux behind
  a thin emit-manifests interface, GitOps-compiler posture (no owned reconcile loop, no
  mutating verbs, no owned state), zero-trust as the differentiator, lakehouse as the
  acceptance workload; refused: multi-runtime, day-2 ops.
- New owner answers are recorded verbatim first, then synthesized; contradictions are
  flagged as findings, not reconciled silently.
- Order of work: owner approves week-one plan → build week-one artifact (compiler + one
  Postgres component via CNPG + mesh mTLS, compiled to Flux manifests) → build out the
  skeleton. Kill review at 4 weeks / 2+ real stacks from the first dogfood week.
- Open item from sign-off: the team name/denominator (Q1.1a) — owner records it before
  the first dogfood week.
- Process enforcement lives in `.claude/` (hooks derive their rosters from
  `docs/README.md` and agent frontmatter — never duplicate a list into a hook).
  Multi-session tasks keep `TASK_PROGRESS.md` at the repo root (agentic-development §4).

## Compact instructions

When compacting, preserve: owner answers/decisions received this session (verbatim),
the status of `00-problem-definition.md` and of the week-one plan approval, any open
contradiction or design question, and TASK_PROGRESS.md step status. Discard exploratory
file-reading history.
