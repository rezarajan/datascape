# Week-four plan — the mesh is the substrate: compiled egress, honest quickstart

**Class: Plan.** **Status: APPROVED as Revision 1 — 2026-07-26, by the owner.
Approval-round answers, verbatim: "Approve as drafted (Recommended)"; deny
breadth "Declared endpoints first (Recommended)" — the namespace-wide deny-all
posture lands as the deliberate next step, exemptions modeled first; manual route
"Companion WALKTHROUGH.md (Recommended)". Product code authorized, scoped to this
revision. Build order: slices 1+2 (compiler, one implementer), then 3 (harness),
then 4 (docs — WALKTHROUGH.md is a new root-level doc, unclassified like README).**

## Ground: two owner directives, recorded verbatim (2026-07-26)

First, on the quickstart: *"A few notes to refine the quickstart: the managed
scenario is not well-defined; it does not include steps like istio installation
and minio install, which prevent it from actually materializing. Each case should
be able to be run in isolation, with clear steps along the way. Furthermore,
simialr to beginner guides, the quickstart should also include a route for the
developer to everything without relying on the nix harness entirely - they need
to understand step-by-step what they are doing, why they are doing it, and why in
that particular order."*

Then, interrupting the steward's mesh-free-managed hypothesis: *"I want to be
clear - the service mesh should be treated as mandatory - a probe to an external
endpoint must go through the service mesh with permission (zero-trust). Omitting
any of these design considerations from the quickstart not only creates confusion
of the very value propositions d7s wants to offer, but it also just does not
deliver on them."*

## Finding (flagged): the plans deferred what the contract commands

Amendment 2 (problem definition, contract): *"Egress is compiled default-deny;
allowlists come only from declared wiring. … v1 = declare + deny."* The week-two
and week-three plans both listed egress compilation as "explicitly NOT this week
(skeleton scope)" — a plan-level deferral in tension with contract text. Scope
authority order resolves this: **the contract wins**; the owner's directive
enforces it. The deferral is superseded, not silently: the managed scenario as
built (no mesh, probe dialing Neon directly) and the self-hosted scenario's
unmediated backup egress to MinIO are both recorded here as contract gaps this
plan closes.

## Slices

1. **Compiled egress: declare + deny, enforced.** For every compiled stack
   namespace the mesh is assumed present (it is mandatory, not per-scenario);
   d7s compiles the egress posture: external and managed endpoints are reachable
   ONLY through the mesh and ONLY by declared wiring — the postgres component's
   backup traffic to its declared `external` store, and declared
   `allowedConsumers` of a managed component to that component's endpoint
   (Istio ambient mechanics — ServiceEntry for the named endpoints, authorization
   scoped to the declared identities, default-deny for everything else — verified
   against current upstream docs at build time, not memory). Everything ships as
   triples: compile-time check, emitted objects, and probes that can fail.
2. **`allowedConsumers` un-refuses on managed placement** — its current refusal
   message promised "enforcement arrives with egress compilation"; it arrives.
   The compiled allowlist gates who may reach Neon through the mesh. The
   `mtls`+`managed` refusal STANDS unchanged (the mesh cannot terminate TLS at a
   provider endpoint — permissioned egress is not mesh mTLS, and no claim
   conflates them).
3. **The managed scenario becomes mesh-mandatory end to end:** istio-install
   joins its orchestrator and prerequisite guards; probe-managed runs as a
   declared consumer identity through the mesh; a negative probe proves an
   undeclared workload CANNOT reach Neon (the egress deny observable live,
   rule 49). The self-hosted scenario gains the equivalent negative probe for
   the external store (undeclared workload cannot reach MinIO).
4. **QUICKSTART rework (first directive):** each scenario is a complete,
   isolated, runnable sequence — no implicit dependencies on the other's steps;
   AND a full manual route with no nix harness at all — the developer drives
   kubectl/flux/istioctl and their OWN git remote (the real GitOps flow needs no
   in-cluster git server), with each step explaining what it does, why it is
   needed, and why it comes in that order. Every documented command actually run
   (the standing rule); the manual route verified by a human-shaped walkthrough,
   not just the harness.

## Explicitly NOT this week (deferred with a home)

Boundary probes beyond egress (import ceremony, v2+); compiled MinIO/object-store
component (skeleton, owner-affirmed at week-three approval); attestation /
declared=running; any new component kind; restore operations (day-2, refused).

## Exit criteria (verified by running — rule 58)

- [ ] Compiled output for both scenarios pins the egress objects in goldens;
      determinism holds; the egress check refuses what declared wiring doesn't
      cover, with remedies (fail-then-pass tested).
- [ ] Live, both scenarios: declared paths work through the mesh; an undeclared
      workload's egress to the external/managed endpoint is REFUSED — observed,
      not assumed.
- [ ] The managed scenario runs in isolation from a fresh cluster with the mesh
      mandatory, via orchestrator AND via its documented piecemeal sequence.
- [ ] QUICKSTART's manual no-harness route runs end to end as written, each step
      with its what/why/order rationale.
- [ ] `allowedConsumers`+managed compiles with enforcement; its old refusal is
      retired with the same discipline the rpo refusal was (goldens, tests,
      harness fixtures updated in the same commit).

## Open questions for the owner (at approval)

- Whether the egress deny for the app namespace should extend to ALL egress
  (strict default-deny per Amendment 2's letter) this week, or only to the
  declared external/managed endpoints' traffic class first (narrower, less risk
  of breaking cluster-internal DNS/API egress; the strict posture lands next).
- Whether the manual QUICKSTART route belongs in QUICKSTART.md itself or as a
  companion (e.g. WALKTHROUGH.md) — one file per audience vs one door.
