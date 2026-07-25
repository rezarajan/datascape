# Week-one artifact plan

**Class: Plan.** **Status: awaiting owner approval — no product code until approved.**

Implements Q4.2 of the signed-off problem definition verbatim: *compiler + one
component + mTLS* — a declaration compiled to Flux-consumable manifests for ONE
component (Postgres via CloudNativePG), deployed with mesh mTLS on. The smallest honest
slice proving **compile-to-secure-running**. The kill-review clock (4 weeks / 2+ real
stacks, Q2.2) starts when this week begins.

## Decisions proposed here (owner confirms or overrides at approval)

1. **Language: Go.** The golden rules' mechanics (optional capability interfaces,
   arch-tested hexagonal layering, single static CLI binary) are written in Go idiom,
   and the platformctl design experience they encode is Go-shaped. Code is NOT reused
   from platformctl — design thinking only (Q1.5).
2. **Mesh: Istio (ambient mode).** Chosen for operational fit (golden rule 62): the
   zero-trust differentiator needs an explicit policy surface (`PeerAuthentication`
   STRICT, `AuthorizationPolicy` allow-lists compiled from declared wiring — rule 53),
   which Istio expresses first-class. *Rejected for now:* Linkerd — simpler and
   mTLS-by-default, but its authorization surface is thinner for the compiled
   least-privilege story. Reversal cost (rule 61): mesh touches only the emitted
   policy objects; a swap re-implements one emitter concern, not the model.
3. **Naming (rule 70):** product name **Datascape** (prose), binary **`d7s`** (what
   operators type), identifier stem **`d7s`** — labels/annotations under
   **`d7s.dev/*`** (e.g. `d7s.dev/managed-by`, ownership per rule 27). The stem
   freezes at first release, not this week, but changing it later is a compatibility
   act — flag now if wrong.
4. **The compiler emits everything an empty cluster needs**: the CNPG operator install
   (Flux `HelmRelease`) as well as the `Cluster` CR — no undocumented prerequisites.
   Flux itself and the mesh are declared environment prerequisites this week (their
   compilation is skeleton work, not week one).

## The acceptance scenario (rule 41 — docs example = e2e test = demo)

```
d7s compile examples/week-one/stack.yaml -o ./out   # deterministic, read-only
git add out/ && git commit && git push               # the plan is the git diff
flux reconcile ...                                   # Flux applies; d7s never does
```
On a kind cluster with Flux + Istio ambient installed:
1. CNPG operator and a 1-instance Postgres come up, all objects labeled `d7s.dev/*`.
2. **Positive probe:** an in-mesh client connects and runs SQL over mesh mTLS.
3. **Negative probe (rule 49):** an off-mesh/plaintext client is REFUSED — observed,
   not assumed; a skipped probe reports `unknown`, never coverage.
4. **Determinism (rules 22/45):** recompiling yields byte-identical output.
5. Removing the component from the declaration and recompiling shows the stated
   removal contract: the Postgres `Cluster` prunes, data PVCs are retained (rules
   28/29 — retain-by-default for data-bearing objects).

## Build order (thin vertical slices — rule 6)

1. **Scaffold:** Go module; hexagonal layout (`internal/domain`, `internal/ports`,
   `internal/adapters/flux`, `cmd/d7s` composition root); arch test enforcing
   inward-pointing imports (rule 8) and a CI fast tier — both proven able to fail
   before first use.
2. **Declaration model:** minimal `Stack` + `postgres` component schema; validation
   aggregates all errors before anything is emitted (rule 33); unknown/reserved
   fields refuse loudly (rule 34); secrets representable only as references
   (rule 51 — week one: a referenced Kubernetes Secret name, never a value).
3. **Compiler core + Flux emitter port:** namespace + operator HelmRelease + CNPG
   Cluster + Flux Kustomization with `dependsOn` from the dependency DAG (rule 24);
   ownership labels on every object (rule 27); golden-file tests from day one.
4. **Zero-trust slice:** STRICT `PeerAuthentication` + default-deny
   `AuthorizationPolicy`, with allows compiled only from declared wiring (rule 53).
   No off switch (rule 50): v1 declarations cannot express "mTLS disabled."
5. **Acceptance harness:** the scenario above scripted end-to-end against kind,
   exercised exactly as an operator would type it (rule 41); both probes executed
   from the real consumer's vantage (rule 30).

## Explicitly NOT this week (deferred with a home — rule 59)

Attestation, admission proofs, and declared=running (skeleton work, Q3.1); the other
skeleton components (Kafka-class, object storage, CDC); compiling Flux/mesh install
themselves; any second GitOps target; self-serve surface. All build out from this
slice toward the Q3.1 skeleton in weeks 2–4.

## Exit criteria (verified by running — rule 58)

- [ ] Acceptance scenario passes on a fresh kind cluster, invoked as documented.
- [ ] Negative probe demonstrably fails when policy is removed (the test can fail).
- [ ] Golden files pin compiled output; two compiles are byte-identical.
- [ ] Arch test and commit-style gate green; both demonstrated able to fail.
- [ ] First dogfood note recorded: time-to-stack measured against the <1 hour target
      (Q2.1), on the owner's real request if one exists this week.

## Open questions for the owner (blocking only where marked)

- **Mesh choice** (blocks step 4): Istio ambient as proposed?
- **Identifier stem/label domain** (non-blocking this week): `d7s`, `d7s.dev/*`?
- **Dogfood substrate** (non-blocking): kind for week one; when does the team's real
  cluster enter?
- **Q1.1a** (contract open item): team name + developer denominator, recorded before
  the first dogfood week.
