# Week-two plan — the managed side of the seam pair

**Class: Plan.** **Status: DRAFT (Revision 0), 2026-07-26 — awaiting owner approval.
No product code lands from this plan until the owner approves it (phase gate).**

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
