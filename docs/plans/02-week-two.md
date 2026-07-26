# Week-two plan — the managed side of the seam pair

**Class: Plan.** **Status: APPROVED as Revision 1 — 2026-07-26, by the owner, with
one owner change (local Neon bootstrap; see "Approval round" below). Product code is
authorized, scoped to this revision's slices and exit criteria.**

Implements the managed half of the contract's single seam pair (amended problem
definition: *placement is a declared binding; v1 proves exactly ONE seam pair —
Postgres: CNPG and managed*). Inputs are the owner decisions of 2026-07-26 (recorded
in `docs/plans/01-week-one.md`): the managed artifact is a **Flux tofu-controller
`Terraform` CR** (single delivery plane), the provider is **Neon** (serverless
Postgres — chosen with its recorded caveat), and this draft precedes any build.

## What the seam proof must show

The same declaration, with only `placement` flipped from `self-hosted` to `managed`,
compiles to a different artifact under the same contract discipline: deterministic,
golden-tested, fail-closed, labels intact. Guarantees behave honestly across the
seam — where a guarantee's meaning cannot survive the placement change, it **refuses
to compile with the remedy**, never silently degrades (rules 34/35/37/50):

- `guarantees.mtls` + `placement: managed` **refuses**: the transport-security family
  is *mesh* mTLS + compiled authorization (problem definition, Amendment 1); a
  provider-terminated TLS endpoint outside the mesh is not that guarantee, and
  pretending equivalence is a best-effort tier. The refusal names the boundary and
  the remedy (self-hosted placement, or drop the guarantee).
- `guarantees.rpo` continues to refuse on both placements (owner decision
  2026-07-26) — Neon's PITR is real but wiring it into the durability triple is
  destination design, deferred with that design.
- Provenance note (Amendment 2): a d7s-compiled managed database is **inside** the
  trust boundary (compiled provenance), not an `external` declaration — `external`
  remains reserved for things d7s never provisions.

## Build order (thin slices, each gated by review)

1. **tofu-controller health check + environment prerequisite.** Pin a version;
   record the project's post-Weaveworks maintenance state as a dated finding. If it
   fails the check (unmaintained, broken on current Flux), the recorded fallback is
   the plain Terraform module (Revision A of the week-one plan) — that swap is an
   owner decision, not a silent adaptation. Controller install joins Flux/Istio as a
   declared environment prerequisite this week (compiling prerequisites is skeleton
   work).
2. **Managed emitter:** `placement: managed` compiles to namespace + `Terraform` CR
   wrapping a Neon provider config (project/branch/database/role), with the Neon API
   key as a referenced Kubernetes Secret only (rule 51 — never a value), ownership
   labels (rule 27), golden files + determinism tests from the first commit (rules
   22/45). Anything the emitter cannot honor refuses with remedy. The
   `planned, not yet available` refusal for `managed` is deleted in the same commit
   its replacement lands.
3. **Guarantee seam behavior:** the `mtls`+`managed` refusal (shown able to fail
   AND pass, rule 49); contract tests pin the messages.
4. **Flux-reconciliation harness wiring** (scheduled by the 2026-07-26 owner
   decision, evidenced by dogfood note 1 finding 4): the harness gains an in-cluster
   git source, Flux consumes the compiled Kustomizations, and the CNPG scenario
   stops direct-applying — the documented `flux reconcile` flow becomes the tested
   flow. The narrowed claims in the week-one plan and README are un-narrowed in the
   same commit, citing the run that earned it.
5. **Acceptance extension:** the documented invocation compiles the managed variant;
   tofu-controller reconciles it on kind; a real Neon database exists afterward; the
   conformance probe connects with the compiled credential reference and runs SQL;
   teardown destroys the Neon resources (a leaked paid resource is a harness
   defect). Negative probes: the `mtls`+`managed` refusal, and the probe's refusal
   to report coverage when skipped (rule 49).

## Explicitly NOT this week (deferred with a home)

RPO on managed placement (durability destination design); compiling egress
default-deny and boundary probes (skeleton, Amendment 2); the import ceremony (v2+);
any second component kind; compiling Flux/mesh/tofu-controller installs; self-serve.

## Exit criteria (verified by running — rule 58)

- [ ] The dogfood declaration with only `placement` flipped compiles to the managed
      artifact; two compiles byte-identical; golden files pin both placements.
- [ ] On a kind cluster: tofu-controller reconciles the emitted CR; a real Neon
      database is provisioned; the probe runs SQL against it using only compiled
      references; harness teardown destroys the Neon resources — verified by
      listing them after.
- [ ] `mtls` + `managed` refuses with the remedy in the error; the refusal is
      demonstrated able to fail and pass.
- [ ] The CNPG acceptance scenario runs through Flux (git source → Kustomizations →
      reconcile), not direct apply; the week-one narrowed claims are un-narrowed.
- [ ] Dogfood note 2 recorded: a second real stack (the kill review needs 2+ by
      2026-08-23).

## Open questions for the owner (at approval)

- **Neon account + API key**: which account/org, and the secret's name — an
  environment prerequisite the operator provides, like the CNPG credentials secret.
- **Teardown policy**: harness runs destroy their Neon resources; does the dogfood
  managed stack (if it becomes dogfood note 2) persist instead, as the real stack's
  home?
- **Does the managed proof double as dogfood note 2**, or does note 2 come from a
  separate real request?

---

## Approval round — 2026-07-26 (answers verbatim, then synthesis → Revision 1)

- **Approval:** "Approve as drafted (Recommended)".
- **Neon secret question** — the owner answered with a directive instead of a name,
  verbatim: *"To avoid incurring costs, bootstrap neon locally instead and simulate
  anything else required to provide the concept around it. This follows the
  principles of 12-factor, where a dev environment is identical in operation to a
  production."*
- **Teardown:** "Harness ephemeral, dogfood persists (Recommended)".
- **Note 2 source:** "Separate real request (Recommended)".

**Finding (flagged, resolved explicitly, not silently):** the first and second
answers conflict — "as drafted" included the exit criterion "a real Neon database is
provisioned" against the paid service, while the Neon answer redirects to a local
bootstrap. The specific answer wins over the general approval; the plan is treated
as **approved with that one change (Revision 1)**, amended here:

- **Slice 1 grows into a feasibility spike:** tofu-controller health check as
  drafted, **plus** the local-Neon path — what Neon ships for local/self-hosted
  operation (e.g. its local docker image), what control-plane API surface the Neon
  Terraform provider needs, and whether that provider can target a local endpoint.
  Whatever the provider cannot do against local Neon is **simulated at the API
  boundary** per the owner directive — but any simulation is named in the record and
  in probe output; a simulated leg is labeled, never passed off as the paid-service
  proof (rule 58's spirit: no claims from memory, none from mocks either).
- **Exit criteria adjusted (Revision 1):** "a real Neon database is provisioned" →
  "a locally-bootstrapped Neon database is provisioned through the same emitted
  `Terraform` CR path, with any simulated control-plane surface named in the harness
  output." Teardown criteria unchanged in substance (leaked local resources are
  still harness defects; cost is no longer the rationale, hygiene is).
- **The API-key secret stays in the design** (the emitted artifact still references
  a secret by name — the seam's shape must not fork between dev and paid; that IS
  the owner's 12-factor point). Default name `neon-api-key` stands unless the owner
  renames it; locally it holds a dummy or local-API token.
- **Evidence caveat for the kill review, recorded now:** the managed seam will be
  proven against a locally-bootstrapped Neon, not the paid endpoint. Dev/prod parity
  is the owner's stated rationale; flipping to the paid endpoint later should be a
  config change, and doing it once before any public claim is the whitepaper rule's
  receipt.

## Slice-1 findings round — 2026-07-26 (→ Revision 2)

Slice 1's feasibility spike (findings recorded in `TASK_PROGRESS.md`, 2026-07-26)
established: tofu-controller is pinnable (moderate risk, OpenTofu-first since
v0.16.0); no local Neon control-plane API exists (the official "Neon Local" image
is a cloud proxy; the open-source local path has no HTTP management API); and every
Neon Terraform provider hard-codes `console.neon.tech`. The Revision 1 directive
("bootstrap neon locally + simulate") therefore costs a fake control plane plus
DNS/TLS interception or a vendored provider fork.

Put back to the owner with those findings; answer recorded verbatim: **"Neon free
tier, real endpoint (Recommended)"** — zero cost via the free tier, the real
provider path, no simulation machinery.

**Finding (flagged): this supersedes Revision 1's local-bootstrap directive** — a
reversal made with fuller information; both texts stand in this record. Revision 2
amendments:

- Exit criteria revert to the real endpoint, at zero cost: **a real Neon free-tier
  database is provisioned** through the emitted `Terraform` CR; the harness requires
  network access and a free-account API key in the `neon-api-key` secret (name
  stands from Revision 1); teardown destroys harness-created Neon resources (free
  tier's project limits make leaks operational defects, not just cost ones).
- The simulation/shim work is dead, not deferred — no home needed.
- Slice 1 is COMPLETE: health verdict recorded (pin an exact tofu-controller tag;
  watch for maintenance lapses; OpenTofu is the engine), local-Neon question closed.
- **Slices 2 and 3 execute as one coherent slice:** removing the `managed` refusal
  while mesh-guarantee declarations could still reach the emitter would create an
  inconsistent fail-open state between commits; the `mtls`+`managed` refusal must
  land in the same commit that opens the managed path (rules 34/37/50). Recorded
  here as a deliberate build-order adjustment, not a silent one.
