# Week-one artifact plan

**Class: Plan.** **Status: Revision A APPROVED — 2026-07-26, by the owner. Istio
ambient confirmed (mesh choice); managed-emit artifact choice deferred to week two
(does not block this build). The original plan below predates Amendment 1 of the
problem definition (guarantees-compiler reframing, 2026-07-25) and is kept unedited;
Revision A at the end adjusts it and is what the owner approved.**

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

- [x] Acceptance scenario passes on a fresh kind cluster, invoked as documented.
      **Verified 2026-07-26** via `scripts/acceptance-kind.sh`, run end to end with no
      manual steps: kind + Flux + Istio ambient + CNPG, the documented `d7s compile`
      invocation, both required negative probes, the durability probe, a positive mTLS
      probe, ephemeral teardown.
- [x] Negative probe demonstrably fails when policy is removed (the test can fail).
      **Verified 2026-07-26** manually against the live cluster: removing the app
      namespace's ambient label let the off-mesh client connect; restoring it restored
      the refusal. Not re-demonstrated on every harness run — proven once, live.
- [x] Golden files pin compiled output; two compiles are byte-identical.
- [x] Arch test and commit-style gate green; both demonstrated able to fail.
- [x] First dogfood note recorded: time-to-stack measured against the <1 hour target
      (Q2.1), on the owner's real request if one exists this week. **Verified
      2026-07-26** — dogfood note 1 in `docs/dogfood.md`: an owner-attested real
      request (internal service/API database), time-to-stack 2m08s vs the <1h target,
      both probes live. Evidence caveats recorded in the note (kind substrate,
      agent-driven operator, placeholder names).

### Environment prerequisites found by running the harness live (2026-07-26)

Not compiled by d7s this week (week-one plan, "explicitly NOT this week") — the
platform team must provide these before applying compiled output:

- **The postgres component's credentials Secret must exist before the Cluster CR is
  applied.** CNPG's `bootstrap.initdb.secret` only consumes a pre-existing secret; it
  does not generate one for a caller-supplied name (`internal/domain/secret.go`).
- **A StorageClass with `reclaimPolicy: Retain`, if golden rule 28's retain-on-delete
  matters for the deployment.** CNPG's `Cluster.spec.storage` has no field of its own
  for this — retention is purely a `StorageClass` property, and d7s cannot safely
  guess a target cluster's CSI provisioner to compile one (golden rule 15). **Owner
  decision, 2026-07-26:** document as a prerequisite rather than add a schema field
  this week; revisit when placement/storage gets dedicated design attention. Verified
  live that the mechanism works when the prerequisite is met (a `Retain`-policy
  StorageClass left the underlying volume `Released`, not deleted, after the Cluster
  CR was removed) — the gap is scope, not a broken mechanism.

## Open questions for the owner (blocking only where marked)

- **Mesh choice** (blocks step 4): Istio ambient as proposed? **RESOLVED 2026-07-26 —
  confirmed at Revision A approval.**
- **Identifier stem/label domain** (non-blocking this week): `d7s`, `d7s.dev/*`? Not
  re-raised at Revision A approval — stands as proposed.
- **Dogfood substrate** (non-blocking): kind for week one; when does the team's real
  cluster enter? Not re-raised at Revision A approval — kind stands for week one.
- **Q1.1a** (contract open item): team name + developer denominator, recorded before
  the first dogfood week. **Still open — not recorded at Revision A approval.**

---

## Revision A — 2026-07-25 (post-Amendment 1: guarantees-compiler)

Everything above stands except as amended here. The slice grows by one declared
guarantee, not by one component — the differentiator is now the guarantee triple, so
week one must prove it.

1. **The declaration gains a `guarantees` block** on the Postgres component. Week one
   implements exactly two guarantee families end-to-end as triples (check → emitted
   infra → conformance probe):
   - *Transport security:* mesh mTLS + default-deny authorization (unchanged from
     steps 4–5 above).
   - *Durability/recovery:* a declared RPO compiles to a CNPG `ScheduledBackup` +
     retention config; the conformance probe verifies a backup object actually appears.
     A declared RPO the emitter cannot honor **fails compilation** with the remedy in
     the error (rules 34/35).
2. **Placement is expressible from day one, compilable to k8s only this week.**
   The component schema carries `placement: self-hosted | managed`; `managed` fails
   closed with "planned, not yet available" (rule 34). The managed side of the seam
   pair (A2) is week-two scope, after the managed-emit artifact decision
   (tf-controller CR / Terraform module / Crossplane claim) is made at this revision's
   approval.
3. **Acceptance scenario gains one probe:** declare RPO on the example stack; verify
   the ScheduledBackup exists and fires once on the kind cluster; then declare an
   unsatisfiable RPO and verify compilation refuses (the negative probe for the
   guarantee primitive — the check must be able to fail, rule 49).
4. **Exit criteria add:** [x] both guarantee triples demonstrated as triples — check,
   emitted infra, and probe each shown working AND shown able to fail. **Verified
   2026-07-26.** Transport security: PeerAuthentication + AuthorizationPolicy emitted;
   positive probe (declared consumer, mTLS) and two negative probes (undeclared
   in-mesh identity; off-mesh plaintext) all live; the refusal shown able to fail (see
   above). Durability: ScheduledBackup emitted from the declared RPO; the probe
   confirms a `Backup` object appears (fires) — it fails with "no barmanObjectStore
   section defined," the documented, deliberate gap since v1 has no object-storage
   component to resolve a destination from (`internal/adapters/flux/durability.go`);
   the unsatisfiable-RPO compile-time refusal is the triple's negative probe and is
   verified live in `scripts/acceptance-kind.sh`.
5. **Open question for approval (blocking week two, not week one):** managed-emit
   artifact for the seam pair — recommendation: Flux tf-controller `Terraform` CR,
   keeping a single delivery plane; plain Terraform module is the fallback if the team
   objects to tf-controller operationally. **DEFERRED 2026-07-26 — the owner declined
   to decide at Revision A approval; this blocks week two only, not the week-one build,
   per the owner's own framing above.**

---

## Owner decisions — 2026-07-26 (steward decision round, post-review findings)

The first steward review pass (recorded in `TASK_PROGRESS.md`, 2026-07-26) found two
places where the artifact's claims outran the contract. Both were put to the owner as
an option round; the selected options are recorded verbatim, then synthesized.

**Q1 — "How should the durability (RPO) guarantee be remedied to satisfy the
contract's 'ships as a triple or not at all' rule?"** Owner selected: **"Fail closed
on rpo (Recommended)"**, whose full option text read: *"`rpo:` refuses to compile in
v1 with the remedy in the error (\"no backup destination declarable yet — planned\").
Emitter, probe, and unsatisfiable-RPO check stay in the tree, gated. Contract-literal
(line 410, rule 34); no amendment needed; plan gets a dated additive correction.
Cost: week one ships one live guarantee triple (mTLS), not two."*

**Q2 — "How should the overstated acceptance claim (Flux never reconciles the emitted
Kustomizations; the harness applies out/ directly) be handled?"** Owner selected:
**"Narrow claim now, wire in week 2 (Recommended)"**, whose full option text read:
*"Dated additive note in the plan + README correction: \"compile + direct-apply
verified live; Flux reconciliation of emitted Kustomizations not yet exercised.\"
Wiring a real git source into the harness becomes a scheduled week-two item, where
the delivery plane matters for the managed seam anyway. Unblocks dogfood
immediately."*

**Synthesis (supersedes the marked items above; earlier text kept, not erased):**

- Revision A item 1's durability triple **does not ship in week one's output**.
  Declaring `guarantees.rpo` now fails compilation closed with the remedy in the
  error, exactly like `placement: managed` (rules 34/35; problem definition line
  408–411). The ScheduledBackup emitter, the durability probe, and the
  RPO-satisfiability check remain in the tree, gated and unit-tested, for the week
  when a backup destination becomes declarable.
- Revision A item 4's exit criterion **"both guarantee triples demonstrated" is
  corrected to: one triple (transport security) demonstrated live in full**; the
  durability family's demonstrated behavior is its compile-time refusal (which now
  covers every `rpo` declaration, not only unsatisfiable targets). The prior checkbox
  text stands as written for what was actually run on 2026-07-26; this note narrows
  what it certifies.
- Exit criterion 1 ("acceptance scenario passes... invoked as documented") is
  **narrowed**: verified live are the documented `d7s compile` invocation, direct
  application of `out/`, the mTLS probes, and CNPG health; **Flux reconciliation of
  the emitted Kustomization objects has not been exercised** (the harness applies
  `out/` directly and says so). Wiring a real git source through Flux is a scheduled
  week-two item alongside the managed-seam work.
