# Week-two plan — the managed side of the seam pair

**Class: Plan.** **Status: APPROVED — 2026-07-26, by the owner; current governing
revision: Revision 4 (branch-per-stack; see the dated revision sections below —
each supersedes what it names and no earlier text is erased). Product code is
authorized, scoped to the current revision's slices and exit criteria.**

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

- [x] The dogfood declaration with only `placement` flipped compiles to the managed
      artifact; two compiles byte-identical; golden files pin both placements.
      **Verified 2026-07-26** — golden fixtures for both placements, determinism
      live in every harness run (Revision 4 shape: branch-per-stack).
- [x] On a kind cluster: tofu-controller reconciles the emitted CR; a real Neon
      database is provisioned; the probe runs SQL against it using only compiled
      references; harness teardown destroys the Neon resources — verified by
      listing them after. **Verified 2026-07-26** — clean `acceptance-managed` run:
      CR Ready, real branch/database/role/endpoint, `SELECT 1` over TLS from the
      written-outputs secret only, in-cluster destroy completed, Neon API confirmed
      no leaked branch. (Ten live attempts to get here; five distinct root causes
      fixed — campaign log in TASK_PROGRESS.md.)
- [x] `mtls` + `managed` refuses with the remedy in the error; the refusal is
      demonstrated able to fail and pass. **Verified 2026-07-26** — contract tests
      fail-then-pass, and the refusal runs live in `compile-and-verify`.
- [x] The CNPG acceptance scenario runs through Flux (git source → Kustomizations →
      reconcile), not direct apply; the week-one narrowed claims are un-narrowed.
      **Verified 2026-07-26** — slice 4 (`3d948b2` + `02246be`), seven-object
      managedFields guard; regression re-passed after slice 5's shared refactors.
- [x] Dogfood note 2 recorded: a second real stack (the kill review needs 2+ by
      2026-08-23). **Recorded 2026-07-26** — the managed case, by owner designation
      (reversal flagged above): `dogfood/managed-api` at 2m53s time-to-stack,
      persisting on `d7s-dogfood` + the Neon project. See `docs/dogfood.md` note 2
      for the four operator-friction findings.

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

## Owner directives — 2026-07-26, mid-slice-4 (→ Revision 3)

Recorded verbatim: *"While that works, two things I want to do to clean up - for
the API key create a process for a developer to safely include that in the test;
this obviously cannot be commited to version control. Ensure as well rational
decisions for CI - you cannot expect a cluster to always be available for CI - a
consequence of the decision to really test on an external service. Second, the
script is becoming very cumbersome. Remember, while agents can easily parse those
scripts it is not easy for a human to do so. I would prefer a fully nixified setup
for all actions; this not only makes things easier to modularize, but genuinely
makes testing deterministic."*

**Synthesis (Revision 3 — two additions, executed in this order after slice 4
clears review; slice numbering continues):**

- **Slice 6 (new, executes before slice 5): fully nixified test/harness actions.**
  The monolithic `scripts/acceptance-kind.sh` decomposes into small, human-readable
  units exposed through the flake (e.g. `nix run .#<action>` via
  `writeShellApplication` or equivalent: cluster-up, flux-install, istio-install,
  git-source, scenario, probes, teardown), each with pinned runtime dependencies
  and shellcheck at build time, composed by a thin orchestrator. Rules carried
  over: no fixed-duration sleeps (44), same entry points locally and in CI,
  ephemeral teardown, probes fail loudly. Determinism rationale is the owner's own:
  pinned inputs make test runs reproducible, and modules make them readable.
- **Slice 5 amendments — API-key process and rational CI tiers:**
  - The Neon API key is NEVER in version control. Local developer process: the key
    enters via the `NEON_API_KEY` environment variable (sourced from the
    developer's own environment or a **gitignored** local file the flake dev shell
    knows how to read); the harness materializes the `neon-api-key` Kubernetes
    Secret from it at runtime, mirroring how the CNPG credentials secret is
    already created per run. The harness refuses the managed scenario loudly, with
    remedy, when the variable is absent.
  - CI tiers (a consequence, owner-named, of really testing an external service):
    the fast tier always runs; the self-hosted kind acceptance runs where Docker
    exists (ubuntu-latest); the **managed/Neon scenario runs only where the
    `NEON_API_KEY` CI secret is available** (push to main / manual dispatch — not
    fork PRs, where GitHub withholds secrets). A skipped managed scenario reports
    itself as SKIPPED/unknown in the job output — never as coverage (rule 49); an
    unreachable external service fails that tier with the remedy, without
    poisoning the tiers that need no network.

## Slice-5 blocker round — 2026-07-26 (→ Revision 4: branch-per-stack)

Slice 5 stopped at the smallest consistent state (§5): the owner's `.env` key is a
**project-scoped** Neon key, which Neon structurally forbids from creating projects
(verified live: `404 project-scoped keys are not allowed to create projects`; the
key is locked to the pre-existing "datascape" project). The approved design
compiled a project per stack. Put to the owner with three options; answer recorded
verbatim: **"Branch-per-stack in your project (Recommended)"**.

**Finding (flagged, resolved explicitly): this reverses the Revision 2 design's
resource set (project-per-stack) with fuller information about the key's scope —
both texts stand.** **Revision 4 synthesis (supersedes the managed emitter's
resource set):**

- **The Neon PROJECT is an environment prerequisite**, exactly like the Kubernetes
  cluster itself — d7s never creates or destroys it. The compiled resource set
  becomes **branch + database + role inside the prerequisite project** (Neon's own
  branch-centric model; least-privilege: CI holds only a project-scoped key,
  never org-wide credentials).
- **The project id never appears in compiled output** — it is an environment
  binding, and baking it in would break determinism across environments (rules
  22/45). It enters at runtime alongside the API key: the `neon-api-key` Secret
  gains a `projectId` key, surfaced to the OpenTofu config as a variable via the
  Terraform CR (`varsFrom`), the same trust path as the key itself.
- **Retain-by-default (rule 28) applies to branches** — they are data-bearing;
  compiled output never enables destruction. Harness teardown patches, deletes,
  and verifies **branch** deletion via the Neon API (a project-scoped key may
  manage branches); a leaked branch is a harness defect.
- The `org_id` provider wrinkle found behind the key blocker is moot (no project
  creation), and the exit criteria's "real Neon database" now reads: a real
  branch + database in the prerequisite project, provisioned and destroyed per
  harness run.

## Owner decisions at slice-5 landing — 2026-07-26

Recorded verbatim (one owner message, on unlocking GPG): *"forego the team name
(it is immaterial to this project), and the second stack will be considered the
managed case."*

- **Dogfood note 2 source: the managed case.** **Finding (flagged): this reverses
  the approval-round answer "Separate real request (Recommended)"** — recorded
  above in this document; both texts stand. Consequence: the managed stack,
  deployed as a persistent dogfood stack (per the approved teardown policy:
  harness ephemeral, dogfood persists), becomes the second real stack for the
  kill review; its demand evidence is the owner's designation rather than a
  distinct request, and the kill review should read it as such.
- The team-name decision belongs to the problem definition (Q1.1a) and is
  recorded there as a struck item in its own contract commit.
