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

## Wildcard-egress finding round — 2026-07-27 (→ Revision 2: exact-host pinning)

Slice 3's live run, the steward's direct experiment, and dispatched upstream
research established (receipts in TASK_PROGRESS and the research record): Istio
1.30 waypoints do not program wildcard TLS ServiceEntries — the capability is
alpha behind the mesh-wide default-off istiod flag
`ENABLE_WILDCARD_HOST_SERVICE_ENTRIES_FOR_TLS`, which upstream marks "not
production ready, susceptible to SNI spoofing, trusted clients only." The
compiled wildcard data-plane edge therefore fails CLOSED (all identities denied
— safe, not correct). The exact-host path is proven live (the provisioner's
console.neon.tech:443 edge works through the same waypoint). Both Postgres
negotiation modes were tested empirically (legacy and sslnegotiation=direct);
the failure is route-absence, not SNI timing.

Owner decision, verbatim: **"Exact-host pinning only"** — whose option text read:
*"No alpha flag. The declaration gains the pinned endpoint host; until pinned,
d7s refuses the consumer data-plane edge with the remedy (provision, read the
endpoint, pin, recompile). Contract-pure and stable-semantics, but first-compile
consumers can't connect through the mesh — a two-step ceremony on every new
managed component."*

**Revision 2 synthesis (supersedes the wildcard data-plane design):**

- The managed component's schema gains an optional pinned endpoint host (the
  value the operator reads from the written-outputs secret after first
  provisioning). `allowedConsumers` on a managed component WITHOUT the pin
  REFUSES compilation, remedy naming the ceremony. With the pin: an exact-host
  ServiceEntry (proven semantics, host-precise — strictly stronger than the
  wildcard's domain precision) + the consumer authorization at 5432.
- The wildcard `*.neon.tech` data-plane ServiceEntry is REMOVED from compiled
  output (no inert or alpha-gated objects). The control-plane edge
  (console.neon.tech:443, tf-runner-only) is unchanged and proven.
- The acceptance harness DEMONSTRATES the documented ceremony live: compile
  without consumers → deliver → read the endpoint host → pin → recompile →
  redeliver → consumer probe through the mesh succeeds; undeclared workload
  refused at the exact-host entry. The ceremony is the documented flow (rule 41),
  not a hidden workaround.
- The alpha flag path is recorded as evaluated and declined (security posture:
  no best-effort tier); revisit only when upstream graduates the feature.

## Enforcement round — 2026-07-27 (→ Revision 3: default-deny pulled forward)

The ceremony run's negative probe FAILED OPEN: the undeclared identity ran SQL
against Neon. Mechanism (structural, evidence in the run log and TASK_PROGRESS):
an identity-scoped ALLOW on a ServiceEntry constrains only traffic routed
through that object; under "declared endpoints first" all other egress from
mesh-enrolled workloads passes as unregistered traffic. The halfway posture
provides no enforceable negative — Amendment 2's compiled default-deny is not
just contract wording but the enforcement mechanism itself.

**Finding (flagged): this reverses the approval round's deny-breadth answer with
live evidence — both texts stand.** Owner decision, verbatim: **"Pull
default-deny forward now (Recommended)"**.

**Revision 3 synthesis:**

- The mesh's outbound policy becomes REGISTRY_ONLY (environment/mesh-install
  configuration — the mesh install remains an environment prerequisite, so this
  lands in the istio-install action and its documentation, not compiled output):
  mesh-enrolled workloads can reach only registered destinations — cluster-local
  services and the compiled ServiceEntries. Unregistered external egress is
  denied at the floor; registered endpoints still enforce identity via the
  compiled authorization.
- Exemption model (deliberate, enumerated): infra namespaces that are NOT
  mesh-enrolled (flux-system, the harness's git/minio scaffolding) are untouched
  by the mesh outbound policy; enrolled namespaces' legitimate egress is exactly
  cluster-local (registered by nature: kube API, DNS, in-mesh services) plus the
  compiled ServiceEntries. Surprises surfaced by the harness are findings.
- The negative probes gain a second leg: an undeclared identity is refused at a
  REGISTERED endpoint (authorization deny) AND at an UNREGISTERED host (registry
  deny) — both observed live, both scenarios.

**Mechanism correction (2026-07-27, same owner decision, evidence-driven):** the
REGISTRY_ONLY mesh-config route is a dead end — maintainer-confirmed
unimplemented in ambient (istio discussion #53021; live-proven here: full HTTPS
round trip to an unregistered host with the config verified present), and
non-enforcing best-effort even in sidecar mode per Istio's own security docs.
The enforceable floor, per upstream's blessed two-layer pattern, is
**Kubernetes NetworkPolicy** — which d7s now COMPILES per enrolled stack
namespace (default-deny egress; allowlist: DNS, istiod control plane,
same-namespace/cluster-internal, HBONE 15008; the waypoint alone granted open
egress as the sole gate) — fulfilling Amendment 2's "egress is compiled
default-deny" more literally than the mesh-config route ever could. Environment
prerequisite: a NetworkPolicy-enforcing CNI (kind ≥ v0.24 enforces natively —
the harness's v0.31 needs nothing; generic clusters need Calico/Cilium in
standard modes, with the known eBPF-mode probe caveats documented). The
community pattern is treated as unverified until our own probes prove it live
(rule 58).

## Control-plane-edge round — 2026-07-27 (→ Revision 4: apiserver edge compiled)

The floor's first live contact (CI runs 30254082510/30270332686 + two local
reproductions; full evidence chain in TASK_PROGRESS, 2026-07-27 sections)
produced two findings:

1. **Revision 3's enforcement premise was half wrong, in both directions.**
   "kind ≥ v0.24 enforces natively" is FALSE for plain pods (probe: full floor
   applied, external egress and apiserver both reachable from a non-ambient
   pod) — but the floor IS enforced for ambient-captured pods, at the mesh
   dataplane (mutation matrix: policies present → every egress attempt from an
   ambient pod dies at the 10s connect deadline; absent → 200 in 5ms; waypoint,
   ServiceEntry, AuthorizationPolicy, Gateway API CRDs each exonerated). Since
   every d7s-compiled namespace with a mesh guarantee is ambient, the floor is
   live exactly where it matters, regardless of CNI — and the CNI prerequisite
   line above overstated what stock kind provides for anything off-mesh.
2. **The floor fails closed against d7s's own operator-managed pods**: CNPG's
   initdb bootstrap needs the kube-apiserver (host-network endpoint — no
   NetworkPolicy pod/namespace selector can name it), so the Cluster never
   leaves "Setting up primary." Same finding class as the tf-runner
   control-plane edge (slice 1): the apiserver edge IS declared wiring implied
   by operator-on-k8s placement.

**Owner decisions, verbatim (2026-07-27, steward question round):** *"Compile
pod+port-scoped edge (Recommended)"* — an egress allowance for TCP 6443+443
scoped by podSelector to the component's own operator-managed pods, implied by
placement, no schema change; precision limit (destination-broad on those ports
for those pods) disclosed, same philosophy as Revision 2's exact-host pinning.
And *"Land now via implementer (Recommended)"* — emitter + goldens + tests land
from the steward session; the held slice-3 WIP rebases on top.

**Revision 4 synthesis:** the NetworkPolicy floor gains one compiled object per
component whose placement implies a control-plane relationship: allow-egress on
TCP 6443 and 443, podSelector-scoped to that component's pods (CNPG instance
pods for self-hosted; the tf-runner pods for managed, extending slice 1's
already-recognized provisioner edge down to the floor layer). Rejected:
schema-knob apiserver endpoint (precise but new per-cluster schema surface —
available later as tightening); CNI-exemption reliance (refuted by evidence —
the mesh dataplane enforces regardless of CNI; assuming exemption is a
best-effort tier, rules 37/50); excluding operator pods from the floor (removes
deny-by-default from the data-bearing pods themselves).

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
