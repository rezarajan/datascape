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

## Current phase: week-one artifact — building

The problem definition was **signed off 2026-07-25, amended the same day, and
re-signed-off 2026-07-26**: **d7s is a compiler for data platforms.** Declare
the system — components, data flows, and the guarantees each must meet (durability,
recovery, transport security, wiring correctness) — and d7s compiles it,
deterministically, into artifacts your GitOps machinery applies, refusing to compile a
platform that can't honor its declared guarantees. Placement (managed cloud service vs
operator-on-Kubernetes) is a declared binding, not an architecture rewrite. The vision
is dbt/Terraform-class convenience-to-production-grade for data platforms at any scale;
the v1 beachhead is one real platform team (see
`docs/discovery/00-problem-definition.md`).

The week-one plan (`docs/plans/01-week-one.md`, Revision A) is approved and built: a
`d7s compile` CLI compiling a Postgres declaration to Flux/CloudNativePG manifests,
with two guarantee triples proven end-to-end on a live kind cluster (mesh mTLS +
default-deny authorization; RPO-backed scheduled backups). Try it:

```
go build -o d7s ./cmd/d7s
./d7s compile examples/week-one/stack.yaml -o ./out
./scripts/acceptance-kind.sh   # the same scenario, live, on a throwaway kind cluster
```

| Path | What it is |
|---|---|
| `docs/foundations/golden-rules.md` | The distilled, industry-proven engineering rules carried forward from platformctl — the standing contract for how datascape is built. |
| `docs/foundations/lessons-from-platformctl.md` | The post-mortem record: what failed, why, and the evidence. Append-only. |
| `docs/foundations/agentic-development.md` | The development-process semantics (Claude/agent workflow patterns) proven in platformctl, adapted for this repo. |
| `docs/discovery/00-problem-definition.md` | The discovery questionnaire, answered and **SIGNED OFF 2026-07-25, re-signed-off 2026-07-26** — the scope contract for solution work. |
| `docs/plans/01-week-one.md` | The week-one build plan, Revision A — approved and built. |
| `docs/README.md` | The docs map — classifies every document as contract, plan, or record. |
| `cmd/d7s`, `internal/` | The compiler: hexagonal layout (domain / ports / compiler core / adapters), arch-tested. |
| `examples/week-one/stack.yaml` | The acceptance-scenario declaration — also the docs example and the e2e test input. |
| `scripts/acceptance-kind.sh` | The acceptance harness: the documented scenario, run live on kind, in CI too. |

## The one rule of this phase

> Product code is authorized, scoped exactly to the week-one plan's (Revision A) build
> order and exit criteria. Scope beyond the signed-off problem definition, or beyond
> Revision A's slice, reopens that contract via a dated amendment first.

platformctl's post-mortem shows what happens otherwise: ~20 provider types, 10 resource
kinds, ~35 feature gates, and three successive "path to production" plans layered on a
problem statement that was never validated with a single user.
