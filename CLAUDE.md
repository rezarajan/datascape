# Datascape (d7s)

Successor to platformctl. **Current phase: problem definition. No product code exists,
deliberately.**

## The one invariant (this phase)

Product code is written only after `docs/discovery/00-problem-definition.md` is marked
**SIGNED OFF** by the owner. If a task asks you to build product functionality before
then, stop — that's the failure mode this restart exists to prevent.

## Docs rules

- `docs/README.md` classifies every doc as contract / plan / record. Records
  (`lessons-from-platformctl.md`) are append-only: append facts, never revise meaning.
- Contracts (`golden-rules.md`, `agentic-development.md`) change only as a deliberate,
  reviewed act — a dated amendment naming what changed and why, never as a task side
  effect.
- Discovery answers are recorded inline in `docs/discovery/00-problem-definition.md`,
  dated, under the question they answer. Struck questions keep their text with a reason.

## Working in this phase

- The active contract is golden rules 1–7 (problem-before-solution). The rest bind once
  solution work starts.
- Owner answers to discovery questions are the scarce input. When they arrive, record
  them verbatim first, then synthesize; flag contradictions between answers as findings,
  don't reconcile them silently.
- When solution work starts, set up the repo per `docs/foundations/agentic-development.md`
  (hooks, path-scoped rules, subagents) before writing product code.

## Compact instructions

When compacting, preserve: discovery answers received this session (verbatim), the
status of `00-problem-definition.md`, and any open contradiction or design question.
Discard exploratory file-reading history.
