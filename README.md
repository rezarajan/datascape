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

## Current phase: problem definition

No product code exists yet, deliberately. The repo currently holds:

| Path | What it is |
|---|---|
| `docs/foundations/golden-rules.md` | The distilled, industry-proven engineering rules carried forward from platformctl — the standing contract for how datascape will be built, whenever building starts. |
| `docs/foundations/lessons-from-platformctl.md` | The post-mortem record: what failed, why, and the evidence. Append-only. |
| `docs/foundations/agentic-development.md` | The development-process semantics (Claude/agent workflow patterns) proven in platformctl, adapted for this repo. |
| `docs/discovery/00-problem-definition.md` | The open questionnaire that must be answered before any solution work begins. |
| `docs/README.md` | The docs map — classifies every document as contract, plan, or record. |

## The one rule of this phase

> Product code is written only after `docs/discovery/00-problem-definition.md` reaches
> **Signed off** status. Everything before that point is problem work, not solution work.

platformctl's post-mortem shows what happens otherwise: ~20 provider types, 10 resource
kinds, ~35 feature gates, and three successive "path to production" plans layered on a
problem statement that was never validated with a single user.
