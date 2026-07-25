# Datascape (d7s)

**Data platform as code — restarted from first principles.**

Datascape is the successor to [platformctl](https://github.com/rezarajan/platformctl), an
exploration of declaring and reconciling data-platform resources the way Terraform and
Kubernetes declare infrastructure. platformctl proved the architecture could be built well —
hexagonal layering held through every audit, the second runtime ran every provider unmodified —
and simultaneously proved that strong architecture cannot rescue an under-defined problem.
Its own planning record ends with the flagship example unable to load a single manifest file.

This repository restarts the quest with the order of operations corrected:
**define the problem first, then build the smallest thing that solves it.**

## Current phase: solution setup

The problem definition was **signed off 2026-07-25 and amended the same day**
(Amendment 1, awaiting re-sign-off): **d7s is a compiler for data platforms.** Declare
the system — components, data flows, and the guarantees each must meet (durability,
recovery, transport security, wiring correctness) — and d7s compiles it,
deterministically, into artifacts your GitOps machinery applies, refusing to compile a
platform that can't honor its declared guarantees. Placement (managed cloud service vs
operator-on-Kubernetes) is a declared binding, not an architecture rewrite. The vision
is dbt/Terraform-class convenience-to-production-grade for data platforms at any scale;
the v1 beachhead is one real platform team (see
`docs/discovery/00-problem-definition.md`). No product code exists yet. The repo
currently holds:

| Path | What it is |
|---|---|
| `docs/foundations/golden-rules.md` | The distilled, industry-proven engineering rules carried forward from platformctl — the standing contract for how datascape will be built, whenever building starts. |
| `docs/foundations/lessons-from-platformctl.md` | The post-mortem record: what failed, why, and the evidence. Append-only. |
| `docs/foundations/agentic-development.md` | The development-process semantics (Claude/agent workflow patterns) proven in platformctl, adapted for this repo. |
| `docs/discovery/00-problem-definition.md` | The discovery questionnaire, answered and **SIGNED OFF 2026-07-25** — now the scope contract for solution work. |
| `docs/README.md` | The docs map — classifies every document as contract, plan, or record. |

## The one rule of this phase

> Product code is written only after the repo is set up per
> `docs/foundations/agentic-development.md` and the golden-rules review (problem
> definition, Q3.4) is done. Scope beyond the signed-off problem definition reopens
> that contract via a dated amendment first.

platformctl's post-mortem shows what happens otherwise: ~20 provider types, 10 resource
kinds, ~35 feature gates, and three successive "path to production" plans layered on a
problem statement that was never validated with a single user.
