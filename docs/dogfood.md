# Dogfood record

**Class: Record (append-only).** Dated notes from real use of d7s, each measured
against the Q2.1 target (declaration to running, verified stack in under one hour)
and feeding the kill review (Q2.2: 4 weeks / 2+ real stacks from the first dogfood
week).

## Dogfood week one — opened 2026-07-26

**Clock:** the first dogfood week starts today, so the kill review falls due
**2026-08-23**, requiring 2+ real stacks by then. (Noted, dormant contradiction: the
week-one plan says the clock "starts when this week [the build week] begins" while
CLAUDE.md and the steward protocol say "from the first dogfood week" — both readings
give the same date because the artifact was built and dogfooding started on the same
day, 2026-07-26. If a future reader needs the clock's authority, this is flagged
here rather than reconciled.)

**Setup decisions (owner, 2026-07-26, steward option rounds — selections verbatim):**

- First stack: **"A real request I have"**, then, asked for the shape:
  **"Internal service/API database"**. The specifics (service name, consumer
  identity, database purpose) were requested in both rounds and not provided; the
  operator run below uses operator-chosen placeholder names, and renaming later is a
  recompile — itself dogfood data. The realness of the demand rests on the owner's
  attestation.
- Substrate: **"kind, locally"** — chosen over a real cluster knowing the evidence
  is weaker (no real workload depends on a local kind cluster); recorded as such for
  the kill review.
- Driver: **"I drive as operator"** — the steward executed the flow. The kill review
  should read every timing below as **agent-assisted operator** evidence: no human
  hand-typed the commands.
- Q1.1a denominator: **5 developers** (recorded in the problem definition,
  2026-07-26; team name still open).

### Dogfood note 1 — internal-api stack, 2026-07-26

**Time-to-stack: 2m08s** from starting the declaration to SQL over mesh mTLS as the
declared consumer, against the Q2.1 target of <1 hour. Environment prerequisites
(kind cluster + Flux + Istio ambient — declared out of d7s scope this week) took a
separate 88s before T0. Stack: `dogfood/internal-api/stack.yaml` (component
`api-db`, consumer `api-server`, mTLS guarantee), compiled to
`dogfood/internal-api/out/` and applied directly; left running on the local kind
cluster `d7s-dogfood` as its home.

Phase timings from T0: compile + infra apply at 16s (includes `go build` of the CLI
itself); CNPG operator ready at 44s; `api-db` "Cluster in healthy state" at 94s;
positive probe (SQL over mTLS as `api-server`) verified at 128s; off-mesh plaintext
client refused as declared.

**Friction observed (each a candidate finding, none acted on silently):**

1. **No installed binary.** The operator ran `go build ./cmd/d7s` first — there is
   no release artifact yet (rule 70's single static CLI has no published build).
   Small today; real the moment anyone but this repo's developers dogfoods.
2. **The credentials-secret prerequisite is manual and order-sensitive.** The
   documented prerequisite (secret before Cluster CR) worked, but the operator must
   remember it unprompted; the compile output does not mention it. An emitted
   post-compile checklist (or the week-two+ secret story) would remove the trap.
3. **The declared consumer's identity is not materialized.** The declaration names
   `api-server` as a consumer, but the ServiceAccount itself had to be created by
   hand before the identity existed to test. Reasonable (the consuming app owns its
   identity) — but nothing tells the operator this; same remedy as (2).
4. **Direct apply makes the operator execute the dependency DAG by hand** (infra →
   wait for operator → secret → app). The emitted Flux Kustomizations encode exactly
   this ordering via `dependsOn` but are not exercised (scheduled week-two work, per
   the 2026-07-26 owner decision) — this run is live evidence for prioritizing that
   wiring.

**Verdict:** the compile-to-secure-running promise held on a real request shape with
96% headroom against the target. Evidence caveats recorded above: kind substrate,
agent-driven operator, placeholder names pending the owner's real ones.

### Dogfood note 2 — managed-api stack (the managed case), 2026-07-26

**Owner designation (verbatim, at slice-5 landing): "the second stack will be
considered the managed case"** — flagged in the week-two plan as reversing the
earlier separate-real-request answer; the kill review should read this stack's
demand evidence as owner-designated, not independently requested.

**Time-to-stack: 2m53s** from starting the declaration to SQL over TLS against a
real Neon database, vs the <1h Q2.1 target. Environment prerequisite (tofu-controller
v0.16.4 onto the existing `d7s-dogfood` cluster, runner image warmed): 34s before
T0. Stack: `dogfood/managed-api/stack.yaml` (component `api-db`,
`placement: managed`, credentials → `api-db-app`), compiled to
`dogfood/managed-api/out/`, delivered through the in-cluster git source + Flux +
tofu-controller — a real branch/database/role/endpoint in the owner's Neon project.
The probe used ONLY the written-outputs secret at the declared `secretRef` name —
zero out-of-band connection values. **The stack persists** (namespace + Terraform CR
on `d7s-dogfood`; branch `api-db` in the Neon project) per the approved teardown
policy: dogfood persists, harness runs stay ephemeral.

**Friction observed (findings, not silently fixed):**

1. **The managed harness actions hardcode the example component name** (`orders-db`
   in deliver/probe/teardown waits) — parameterized for namespace/stack/out but not
   component, so the operator hand-drove delivery with kubectl/flux instead of
   reusing `deliver-managed`/`probe-managed`. Harness ≠ operator tooling yet.
2. **GitRepository registration is inside a harness action, not the documented
   operator flow or compiled output.** The operator must know the git server's
   service URL and namespace out of band (and the namespace differs from the
   deployment's name — `d7s-harness-git` vs `d7s-gitserver` — which cost a failed
   attempt). The documented flow's "git push; flux reconcile" needs the
   GitRepository named as an explicit environment prerequisite.
3. **The `projectId` secret key is hand-supplied in the operator flow** — the
   harness self-discovers it from the key's scope error, the operator has no
   equivalent affordance.
4. **PodSecurity warning** on the `tf-runner-warm` pod (violates `restricted`
   profile) — cosmetic on kind, a real blocker on a restricted-enforcing cluster.

**Verdict:** the seam holds as a dogfood stack — same declaration shape, placement
flipped, real managed database running with compiler-wired credentials, at 95%
headroom against the target. Kill-review status: **two stacks recorded** (caveats:
both kind-substrate, both agent-driven, note 2's demand owner-designated).
