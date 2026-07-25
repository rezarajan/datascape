# Datascape (d7s)

Successor to platformctl. **Problem definition SIGNED OFF 2026-07-25** (see
`docs/discovery/00-problem-definition.md` — now a contract). **Current phase: solution
setup. No product code exists yet.**

## The one invariant (this phase)

Product code is written only after solution setup is complete: (1) the repo is set up
per `docs/foundations/agentic-development.md` (hooks, path-scoped rules, subagents), and
(2) the golden-rules review recorded at Q3.4 of the problem definition is done. If a
task asks you to build product functionality before then, stop. Scope beyond the
signed-off problem definition reopens that contract via dated amendment first.

## Docs rules

- `docs/README.md` classifies every doc as contract / plan / record. Records
  (`lessons-from-platformctl.md`) are append-only: append facts, never revise meaning.
- Contracts (`golden-rules.md`, `agentic-development.md`) change only as a deliberate,
  reviewed act — a dated amendment naming what changed and why, never as a task side
  effect.
- Discovery answers are recorded inline in `docs/discovery/00-problem-definition.md`,
  dated, under the question they answer. Struck questions keep their text with a reason.

## Working in this phase

- All golden rules now bind (solution work has been authorized), pending the Q3.4 review
  that may strike specific rules with recorded reasons.
- The signed-off problem definition is the scope authority: one k8s target, Flux behind
  a thin interface, GitOps-compiler posture (no owned reconcile loop), no multi-runtime,
  no day-2 ops. Zero-trust and the lakehouse workload are IN scope by owner decision.
- New owner answers or scope decisions are still recorded verbatim first, then
  synthesized; contradictions are flagged as findings, not reconciled silently.
- Order of work: repo setup per `docs/foundations/agentic-development.md` + golden-rules
  review (Q3.4) → week-one artifact (compiler + one component + mTLS) → build out the
  skeleton. Kill review at 4 weeks / 2+ real stacks.

## Compact instructions

When compacting, preserve: discovery answers received this session (verbatim), the
status of `00-problem-definition.md`, and any open contradiction or design question.
Discard exploratory file-reading history.
