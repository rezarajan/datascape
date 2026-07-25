# Lessons from platformctl

**Class: Record (append-only).**

This is the post-mortem of the platformctl project — the predecessor to datascape —
distilled from its own planning documents (docs 07–12), bug ledgers, ADR amendments, and
remediation findings. It exists so the restart never argues from memory about why the
restart happened.

## The verdict

platformctl was an architecturally strong solution to an under-defined problem.

Both halves of that sentence are backed by its own record:

- **Architecturally strong:** its final systems review concluded "the hexagonal
  architecture is genuinely load-bearing" — layering held through every audit, and the
  second runtime (Kubernetes) ran every provider unmodified. The engineering practices in
  `golden-rules.md` are real and worked.
- **Under-defined problem:** the product requirements asserted "there is no equivalent of
  a Kubernetes manifest for the data platform" and never validated it — no user
  interviews, no competitive analysis against Compose/Testcontainers/Tilt/Helm, no named
  users. All success criteria were mechanism-facing (idempotency, drift) rather than
  outcome-facing (time-to-value, adoption). Three personas pulled in different directions,
  and the vaguest one (platform-substrate for an IDP) silently justified most of the
  architectural weight.

When an agent finally attempted to *use* the tool to build a real platform, it hit issues
at every turn — culminating in the terminal finding of the project's own task log
(doc 08 §10, 2026-07-24): **the flagship 26-resource example could not be applied at all**,
because the manifest loader had never implemented directory recursion. The product's most
basic documented invocation had never been true.

## The numbers

| Signal | Value |
|---|---|
| Resource kinds | 10 shipped (+9 retired experiments, +9 deferred) |
| Provider types | ~20 |
| Feature gates | ~35 |
| CLI subcommands | 11+ |
| Successive "path to production" planning regimes | 5 (docs 07, 08, 09, 11, 12) — each defining "production" more broadly than the last |
| Defects found **only** by live testing | 17 (13 Kubernetes + 4 Docker), reducing to 5 systemic classes |
| Call sites carrying one identical bug (hardcoded addresses) | 10 |
| Remediation findings from auditing claimed status vs code | 10 (F-001…F-010), incl. a checked stage-gate box that was false |
| Anonymous volumes leaked by an unstated cleanup contract | 3,853 (8.4 GB) |
| Remaining tasks in the *final* plan when the project stopped | 34, across 6 phases — beginning with formal verification that had not started |

The discovery rate of serious defects never converged toward zero: a single day of
production review (2026-07-22) on a fully-green codebase produced a confirmed shell
injection, four async-correctness findings, ~7 root-cause fixes, and three sequenced GA
blockers.

## The five failure classes (from the live-testing ledger, doc 09)

All 17 live-only defects reduce to five classes — each one "fixed multiple times, at
multiple independent call sites, by different sessions: the definition of a missing
system-level mechanism."

1. **Network topology leaked into dependents.** Ten call sites constructed
   `"127.0.0.1:" + port` instead of resolving observed addresses; correct on Docker only
   by coincidence. → *Never construct what you can resolve.*
2. **exists ≠ ready ≠ reachable conflated.** Three states collapsed into one; every
   conflation produced a race fixed by yet another hand-rolled retry loop. → *Readiness
   semantics belong in the contract.*
3. **Under-declared intent that a permissive runtime tolerates.** Docker forgives;
   Kubernetes interprets literally. → *The most permissive backend must never define the
   contract; the fake enforces the strictest reading.*
4. **Runtime-object identity by convention.** Consumers re-derived names from unwritten
   conventions; the identical mistake was made and fixed twice. → *One naming authority;
   consumers resolve published facts.*
5. **Contract tests prove the port, not the translation.** Conformance stayed green while
   the Kubernetes adapter replaced every image's entrypoint — the synthetic test image had
   no entrypoint to notice. → *Real workloads on real infrastructure are the acceptance
   bar; every live bug back-fills a contract-level reproduction.*

The synthesis finding: the connectivity/discovery plane **was never named as a layer**, so
its logic precipitated into whichever provider needed it that day — exactly where 10 of
the 17 defects lived. *The unnamed plane becomes the smeared responsibility.*

## Failures of claim integrity

A recurring meta-class, distinct from code bugs: **the record said things the code didn't do.**

- `metadata.protect` (the anti-deletion safety marker) was inert from the day it shipped —
  the real manifest loader never decoded it, while engine tests stayed green by
  hand-constructing internal values.
- Network isolation was never enforced in CI — the test cluster's CNI never enforced
  NetworkPolicy, so every isolation assertion silently skipped for the project's life.
- A backup pipeline reported success through five layers while writing a 0-byte file with
  a matching empty checksum.
- `sslmode=disable` was hardcoded in every database-facing consumer, making the flagship
  production scenario (cloud-managed databases) structurally impossible while the docs
  claimed production readiness.
- `Connection.spec.via` (the VPC/VPN path) was schema-accepted and consumed by nothing — a
  silent security no-op.
- Stage-gate checkboxes were found checked-but-false (and fixed-but-unchecked) four times.

None of these were exotic: each was a claim that was never verified through the real
end-to-end path from the user's side of the interface.

## Process lessons

- **Model-shape mistakes required owner intervention, not process detection.** The
  binding-taxonomy revision (function → relation) and the Catalog/Connection remodel both
  happened because the owner noticed, mid-flight. Shape review must precede shipping
  shapes, because shapes are the compatibility contract.
- **Two correct features were mutually destructive** (default-deny isolation silently
  blocked the access-mode feature). Composition needs its own acceptance scenarios.
- **The first root-cause analysis of the worst bug was wrong**, and was only corrected by
  direct reproduction. Reproduce, then conclude; record disproven hypotheses.
- **Definition-of-production drift:** five successive planning documents each widened the
  bar (demo gaps → production contract → segregation readiness → "usable for production" →
  "tech firms rely on it"). The bar must be fixed before the claim, not discovered after it.
- **What worked and should be reused:** the agentic development machinery (tiny CLAUDE.md,
  path-scoped rules, enforcement hooks, cost-tiered subagents, TASK_PROGRESS resumability,
  the test-impact ledger) demonstrably caught and prevented real classes of error — see
  `agentic-development.md`.

## What this means for datascape

1. The problem gets defined, validated, and signed off before solution work
   (`docs/discovery/00-problem-definition.md`).
2. Scope is a budget, not an aspiration: one persona, one job-to-be-done, one worked
   scenario that is exercised exactly the way a user would invoke it, from the first week.
3. The golden rules carry forward as the engineering contract — they were never the
   problem. The problem was what they were aimed at.
