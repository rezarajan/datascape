# TASK_PROGRESS — solution setup + week-one plan

Resumability file per `docs/foundations/agentic-development.md` §4. Branch: `main`
(commits land directly here this session; no feature branch in use). A session
resuming this task needs only this file plus `git log`.

## Step plan and status

1. **Golden-rules review (Q3.4)** — COMPLETE (commit `50c275f`). 67/70 rules bind;
   21/25/26 struck with reopen criteria; 8 compiler-shape interpretations. Recorded as
   the dated 2026-07-25 amendment in `docs/foundations/golden-rules.md` and under Q3.4
   in the problem definition. Q3.4 sign-off open item is closed.
2. **Repo setup per agentic-development.md** — COMPLETE (this commit). Hooks with
   repo-relative paths (`$CLAUDE_PROJECT_DIR`); classified-docs guard derives its
   protected set from `docs/README.md`; subagent-model guard derives its roster from
   `.claude/agents/` frontmatter (both platformctl drift defects fixed);
   doc-consolidation cadence (28d) with SessionStart reminder and
   `docs/consolidation.md` record; path-scoped rules (`.claude/rules/`); three
   checked-in subagents (test-runner, manifest-verifier: haiku; contract-reviewer:
   sonnet). Every gate demonstrated capable of failing AND passing (15/15 checks,
   run 2026-07-25).
3. **Commit-style enforcement (owner directive, 2026-07-25)** — COMPLETE. Strict
   google/conventional style gated by `.githooks/commit-msg` (activated via
   `core.hooksPath` by a SessionStart hook); gate demonstrated failing and passing
   (10/10 checks). The two unpushed commits were reworded to conform. The four
   pre-existing commits (`7c38429..c24f4ec`, shared with `origin/main`) keep their
   messages — rewriting shared history is an owner decision, not taken.
4. **Week-one plan** — WRITTEN at `docs/plans/01-week-one.md`; status: **APPROVED
   2026-07-26** (Revision A). Istio ambient confirmed; managed-emit artifact deferred
   to week two.
5. **Problem-definition re-sign-off** — COMPLETE (commit `9b8f716`, 2026-07-26).
   Amendment 1 (guarantees-compiler) and Amendment 2 (trust-boundary) both
   re-signed-off in one owner confirmation; also fixed an inconsistent prior partial
   flip (commit `1248528` had updated only the top status line, not the amendment
   footers or `docs/README.md`).
6. **Week-one build** — IN PROGRESS.
   - (a) Go module + hexagonal scaffold + arch test — COMPLETE (`a75e6e7`). Arch test
     verified able to fail on 3 distinct violations and pass cleanly reverted.
   - (b) Stack+postgres declaration schema (guarantees + placement) — COMPLETE
     (`605fda3`). YAML loader with strict unknown-field rejection; validation
     aggregates all errors; `placement: managed` fails closed.
   - (c) Compiler core + Flux emitter (namespace, CNPG operator HelmRelease, Cluster
     CR, Kustomization `dependsOn`, `d7s.dev/*` labels) — COMPLETE (`00bface`).
     Golden-file + determinism tests verified able to fail and pass. CLI's
     `d7s compile <file> -o <dir>` verified against the exact documented invocation.
   - (d) Zero-trust slice (STRICT PeerAuthentication + default-deny
     AuthorizationPolicy from declared `allowedConsumers`) — COMPLETE (`afb5215`).
   - (e) Durability slice (RPO → CNPG ScheduledBackup; unsatisfiable RPO refuses with
     remedy) — COMPLETE (`6f7b1cc`). No `barmanObjectStore` destination is emitted —
     v1 has no object-storage component/`external` declaration to resolve one from;
     flagged as a finding for the live acceptance run.
   - (f) Acceptance harness on kind — **COMPLETE** (`4b2a72c`, `a2afe97`). Was blocked,
     then unblocked — see findings below. `scripts/acceptance-kind.sh` runs the full
     documented scenario end to end with no manual steps and ephemeral teardown;
     wired into CI as a separate deep-tier job (`.github/workflows/ci.yml`). All
     Revision A exit criteria checked off in `docs/plans/01-week-one.md` except the
     dogfood-timing note, which needs a real owner request.

## Finding: kind/minikube cannot run in this sandbox (2026-07-26)

Both `kind create cluster` (bridge network, and again forcing `--network=host`) and
`minikube start` (docker driver, pre-existing stopped profile) fail identically:
`failed to set up container networking: ... operation not supported` when Docker
tries to create a veth pair for the node container. This is a sandbox-level
restriction on nested network-namespace/veth creation, not a config problem — cleaned
up after each attempt (no leftover containers/networks beyond what pre-existed).

The only other viable path is minikube's `--driver=none`, which runs kubelet/kubeadm
directly on the host with no container isolation — a materially more invasive,
harder-to-reverse action than anything else this task has done, so it was **not**
attempted without explicit authorization.

**Consequence:** exit criteria items requiring a running artifact (acceptance scenario
on kind, both negative probes, the durability probe firing a real backup, the
`barmanObjectStore`-less Cluster/ScheduledBackup shape actually being accepted by
CNPG's admission webhook) are **unverified**. Everything gated on `go test`/`go build`
(scaffold, schema, compiler+emitter, golden files, determinism, zero-trust and
durability compile-time checks) is verified live; everything requiring a live
Kubernetes API is not. Reported to the owner rather than silently adapting scope
(agentic-development §5) or marking exit criteria done from memory (rule 58).

## Finding correction: veth failure is a pending-reboot kernel issue, not a sandbox restriction (2026-07-26)

Re-diagnosed with the owner. The `operation not supported` on veth-pair creation
reproduces **outside** the command sandbox (`docker run --rm alpine ip link` fails
identically against the host daemon), so it is not a sandbox restriction. Root cause:
the host is running kernel `7.1.3-2-cachyos`, but `/lib/modules/` contains only
`6.18.38-2-cachyos-lts` and `7.1.4-1-cachyos` — the kernel package was upgraded after
boot, deleting the running kernel's module tree. The `veth` module was not loaded at
the time (`modinfo veth`: module not found) and can no longer be loaded, so Docker
cannot create container network endpoints. **Resolution: reboot the host** (into
7.1.4), then re-verify with `docker run --rm alpine ip link`.

Two consequences for the earlier finding:

- `minikube --driver=none` would **also** have failed — pod networking via CNI creates
  veth pairs too. Not attempting it was correct, and it is off the table regardless.
- No sandbox constraint exists, so the harness needs no exotic substrate.

**Decision for task 8 (owner-directed re-plan): kind, locally and in CI, same
script.** Rationale recorded: kind is conformant Kubernetes in Docker and is what
CNPG's and Flux's own test suites run on; it runs unchanged on GitHub Actions
`ubuntu-latest` (native Docker, no nested virtualization); node image pinning gives a
reproducible cluster version; create/teardown per run keeps it ephemeral.
Vagrant+Talos was considered and rejected for this phase: it needs KVM/libvirt
locally and in CI, is slower and heavier, and buys nothing while the guarantees under
test (mTLS wiring, ScheduledBackup firing, CNPG admission of the
`barmanObjectStore`-less shape) are Kubernetes-API-level, not host-OS-level.
Note: `kind` is not yet installed (nix profile has minikube/kubectl/flux) — add it
alongside the harness.

## Finding resolution: acceptance harness green, two live-caught composition bugs fixed (2026-07-26)

kind (v1.36.1 node image), Flux, and Istio ambient all installed cleanly once the
kernel issue was resolved. Running the full scenario live surfaced three real
findings, none of them visible from `go test` alone (golden rule 40):

1. **Credentials secret must pre-exist.** CNPG's `bootstrap.initdb.secret` only
   consumes a pre-existing Secret; it does not generate one for a caller-supplied
   name. The role/database were created with no password until the secret existed
   first. Documented in `internal/domain/secret.go` and as a harness prerequisite.
2. **AuthorizationPolicy blocked CNPG's own operator traffic** (commit `4b2a72c`).
   The unscoped allow-list also gated the operator's instance-status polling
   (port 8000), not just Postgres (5432) — a rule-42 composition bug (a security
   feature and a durability feature were mutually destructive). Fixed by scoping
   consumer rules to port 5432 and adding a separate operator-namespace rule scoped
   to port 8000 only.
3. **The operator's own namespace needed the ambient label too.** Scoping the
   AuthorizationPolicy alone didn't fix it: PeerAuthentication enforces STRICT mTLS at
   the transport layer, before AuthorizationPolicy is ever evaluated, so a client
   outside the ambient mesh can't originate the connection regardless of policy.
   `cnpg-system` now joins the ambient mesh whenever any component declares `mtls`.
4. **PVC retention-on-delete needs a `Retain`-policy StorageClass** — CNPG has no
   field of its own for this; it's purely a `StorageClass` property, and d7s can't
   safely guess a real cluster's CSI provisioner to compile one (rule 15). Verified
   live that the mechanism works when the prerequisite is met (PV left `Released`,
   not deleted). **Owner decision, 2026-07-26:** document as an environment
   prerequisite this week rather than add schema scope; revisit with placement/storage
   design later. Recorded in `docs/plans/01-week-one.md`.

After all three fixes, one clean run of `scripts/acceptance-kind.sh` passes every
check together: positive mTLS probe, an undeclared in-mesh identity refused, an
off-mesh plaintext client refused, the CNPG operator still reaching "Cluster in
healthy state", the unsatisfiable-RPO compile-time refusal, determinism, and the
durability probe (a `Backup` object appears, failing only on the documented,
deferred object-storage gap). Golden files, tests, and code comments updated to
match; commits `4b2a72c` and `a2afe97`.

**Process note:** mid-session, GPG commit signing hung on pinentry (gpg-agent
passphrase cache expired) — held rather than bypassing with `--no-gpg-sign`; the
owner unlocked their agent out of band and signing resumed normally.

## Finding + fix: harness toolchain was unpinned; now provided by flake.nix (2026-07-26)

Caught during the first steward review pass: re-running `scripts/acceptance-kind.sh`
failed at `istioctl: command not found` — the binary used for the green 2026-07-26 run
was no longer on PATH — and the CI acceptance job never installed the `flux` CLI at
all (absent from `ubuntu-latest`), so it could not have passed as written; `istioctl`
there came unpinned from `curl | sh`. Owner directive: define every required test
binary in a `flake.nix` for consistency across environments. Done — dev shell pins
go/kind/kubectl/fluxcd/istioctl/openssl; CI's acceptance job now enters through
`nix develop`; the fast tier stays on `setup-go` (go.mod already pins its only tool,
and nix install overhead would eat the 5-minute budget that guards golden rule 43).
The full harness was then re-run through the flake: **all seven checks pass**
(compile, determinism, unsatisfiable-RPO refusal, cluster healthy, positive mTLS
probe, off-mesh refusal, durability probe firing) with clean teardown. CI-side run
still unverified until the owner pushes.

## Owner decisions: rpo fails closed; Flux-path claim narrowed (2026-07-26)

Steward decision round on the two review findings (options + verbatim selections
recorded in `docs/plans/01-week-one.md`, "Owner decisions — 2026-07-26"):

1. **Durability:** `guarantees.rpo` fails compilation closed in v1 (no backup
   destination is declarable, so the triple's probe leg could never pass — problem
   definition line 408–411). Emitter/probe/satisfiability check stay gated in the
   tree. Week one ships one live triple (mTLS).
2. **Acceptance claim:** narrowed now (plan + README); wiring Flux to reconcile the
   emitted Kustomizations from a real git source is scheduled week-two work.

Execution: implementer slice dispatched for the fail-closed change + README
correction; `.claude/agents/implementer.md` added (the steward protocol's implementer
tier had no definition — process gap found and closed this session).

**Slice COMPLETE (2026-07-26):** `0333800` (rpo fails closed in the domain layer,
aggregating with other errors; gated emitter/probe machinery kept unit-tested;
example + goldens regenerated; harness probe retargeted; README narrowed) +
`9c8e5f4` (review follow-up: the rpo refusal's aggregation with other validation
errors pinned by a dedicated test — the one contract-review finding). Verified:
fast tier green, and the full acceptance harness re-run green through the flake
(compile, determinism, any-rpo refusal, cluster healthy, positive mTLS probe,
off-mesh refusal, clean teardown). Both review findings from the first steward
pass are now closed; the Flux-reconciliation harness wiring is scheduled week-two
work per the owner decision.

## Dogfood week opened; first note recorded (2026-07-26)

Owner directed the dogfood start (setup selections recorded verbatim in
`docs/dogfood.md`). Q1.1a denominator (5) recorded in the problem definition via the
unlock protocol; team name still open. Dogfood note 1: internal-api stack
(owner-attested real request, placeholder names), **time-to-stack 2m08s vs the <1h
Q2.1 target**, positive mTLS probe + off-mesh refusal both live; stack left running
on the local `d7s-dogfood` kind cluster. Four friction findings recorded in the note
(no release binary; manual secret prereq; consumer identity not materialized; direct
apply executes the DAG by hand — live evidence for the scheduled week-two Flux
wiring). Last Revision A exit criterion checked off. Kill review due **2026-08-23**.

## Managed-emit decided; week-two plan drafted (2026-07-26)

Owner decision round (verbatim selections recorded in `docs/plans/01-week-one.md`):
managed emit = **tofu-controller `Terraform` CR**; seam provider = **Neon** (with
its recorded caveat); week-two planning = draft now. `docs/plans/02-week-two.md`
created as **DRAFT Revision 0, awaiting owner approval** — no product code from it
until approved. Key plan shape: mtls+managed refuses (mesh guarantee doesn't cross
the seam), rpo still refuses everywhere, Flux-path harness wiring un-narrows the
week-one claims, Neon teardown verified in the harness.

## Week-two slice 1 (feasibility spike) — findings, 2026-07-26

Research verified against upstream sources (releases, source code) — dated facts:

- **tofu-controller (flux-iac):** revived and active — v0.16.4 (2026-06-08), monthly
  releases since v0.16.0 (2026-01-27), issue backlog shrinking; BUT OpenTofu-first
  since v0.16.0 (breaking; OpenTofu bundled by default), community-maintained with a
  prior ~2.5-year stable-release drought (v0.15.1 Jul 2023 → v0.16.0 Jan 2026), and
  no documented compatibility matrix against current Flux 2.9. **Health verdict:
  pinnable at an exact tag as a moderate-risk environment prerequisite.**
- **Local Neon:** the official "Neon Local" docker image is a proxy to the paid
  cloud (requires NEON_API_KEY + project id — no account, no function). The genuine
  no-account path is the open-source repo's control_plane crate (`cargo neon`:
  tenants/timelines/endpoints) or its docker-compose storage stack — neither exposes
  an HTTP control-plane API shaped like the cloud's `console.neon.tech/api/v2`.
- **Neon Terraform providers:** community-only (kislerdm/neon ~584k downloads;
  terraform-community-providers/neon), and both **hard-code
  `console.neon.tech`** with no endpoint override (confirmed in provider/SDK
  source; the SDK's `baseURL` is a constant, the community provider sets
  `req.URL.Host` unconditionally).
- **Consequence:** "simulate anything else required" (owner directive, Revision 1)
  means building a fake Neon REST control plane implementing the project/branch/
  database/role/endpoint CRUD surface PLUS DNS+TLS interception inside runner pods,
  or forking/vendoring the provider SDK. Nontrivial standing machinery that proves
  the provider can talk to a fake — a deviation-scale finding per §5, stopped and
  reported to the owner rather than built silently.

## Week-two slices 2+3 (managed emitter + seam refusals) — built, pending commit (2026-07-26)

Implementer slice under plan Revision 2: `placement: managed` compiles to a pinned
`kislerdm/neon` (0.14.0) OpenTofu config (project/role/database) plus a
tofu-controller `Terraform` CR (`infra.contrib.fluxcd.io/v1alpha2`, verified
upstream, `approvePlan: auto`), API key by `neon-api-key` secret reference only;
mtls+managed and allowedConsumers+managed refuse with boundary + remedy; rpo still
refuses everywhere; managed golden fixture + determinism + refusal contract tests;
self-hosted output proven byte-identical (no regression). Review: one finding —
the managed credentials gap (`credentials.secretRef` declared but not consumed;
Neon role created without password, no app-credentials Secret emitted) was
disclosed only in the agent's report, not the artifact (rule 34) — returned for a
dated in-code deferral note pointing at slice 5, where it must be wired
(tofu-controller `writeOutputsToSecret` is the expected mechanism) or explicitly
re-deferred. Test tiers green (build/vet/test, determinism both placements, three
refusal spot checks). Implementer-flagged naming choice for owner: the key inside
the `neon-api-key` Secret is `apiKey`.

**Blocker: GPG signing.** gpg-agent's passphrase cache expired mid-session;
pinentry times out unattended (same incident class as the recorded week-one one).
The slice commit and this record's commit are held unsigned-nothing-bypassed until
the owner unlocks the agent.

## Owner directives mid-slice-4: API-key process, rational CI, nixified harness (2026-07-26)

Recorded verbatim in `docs/plans/02-week-two.md` (Revision 3). Execution order once
slice 4 clears review (overlapping surface — no parallel implementers on the
harness): **slice 6 first** (decompose the monolithic harness into flake-exposed,
shellcheck-gated, human-readable `nix run .#<action>` units), **then slice 5** on
that base (managed/Neon acceptance with `NEON_API_KEY` entering only via
environment/gitignored local file → runtime Secret; CI runs the managed tier only
where the secret exists, reporting SKIPPED/unknown otherwise, never coverage).

## Week-two slice 4 (Flux-path harness) — landed, fix round in progress (2026-07-26)

Commit `3d948b2`: the harness now stands up a harness-lifetime in-cluster git
server (debian-slim + nginx + fcgiwrap, smart HTTP — Flux's GitRepository schema
accepts only http/https/ssh, and Alpine ships no git-http-backend), pushes compiled
`out/` to it, applies ONLY the GitRepository + the two emitted Kustomization CRs,
and lets kustomize/helm-controller deliver everything else; a managedFields guard
fails loudly on hand-applied compiled objects. Full live pass verified from the run
log; no emitter changes were needed (the emitted sourceRef/path shape already
matched). Claims un-narrowed additively in the week-one plan + README, citing the
run. **Implementer-flagged emitter gap:** emitted Kustomizations carry no
`spec.healthChecks`, so `dependsOn` alone doesn't gate on CNPG operator readiness —
harness compensates with an explicit wait + Flux's on-demand reconcile verb;
emitter-side fix is a named future-slice candidate. **Review findings (returned for
one fix commit + mandatory live re-run):** (1) the guard claimed all compiled
objects but covered 3 of 7 — the zero-trust pair among the unguarded; (2) the
healthChecks gap was script-comment-only while the plan note claimed flat closure.

**Slice 4 ACCEPTED (2026-07-26):** fix commit `02246be` — guard extended to all
seven compiled objects (both zero-trust objects included), healthChecks gap
disclosed with a dated line in the week-one plan; full harness re-run PASSED with
the extended guard (verified from the run log), fast tier green.

## Week-two slice 6 (nixified harness) — ACCEPTED (2026-07-26)

Commit `34fa275`: the monolith `scripts/acceptance-kind.sh` is deleted, replaced by
ten single-purpose flake actions (`scripts/actions/*.sh` + shared
`scripts/lib/common.sh`, one bounded-poll definition) built with
`writeShellApplication` — shellcheck + pinned runtimeInputs per unit — composed by
`nix run .#acceptance`; CI and README point at the same entry point. Verified live:
full scenario passed via the new entry point (compile, determinism, any-rpo
refusal, git-source, Flux-only delivery, seven-object guard, both probes, clean
teardown), `nix flake check` green, fast tier green. Shellcheck immediately caught
one real pre-existing bug in carried-over guard code (unquoted jsonpath). Contract
review: NO findings — behavior parity confirmed check-by-check against the deleted
monolith. Remaining week-two work: slice 5 (managed/Neon acceptance + credentials
wiring via writeOutputsToSecret or explicit re-deferral + API-key process + CI
managed-tier gating) — blocked on the owner's NEON_API_KEY.

## Week-two slice 5 — live verification campaign log (2026-07-26, ongoing)

The managed scenario's live runs surfaced five distinct real-world defects, none
visible from `go test` (golden rule 40), each diagnosed from captured evidence and
fixed in turn: (1) harness ordering — tofu-controller installed before `flux
install` created its target namespace; (2) `kind load docker-image` fails on
multi-arch ctr import — replaced with a kubelet-driven image warm (disposable pod);
(3) tofu-controller v0.16 disables cross-namespace source refs by default —
steward decision: enable at install (matches the cluster's kustomize-controller
posture; full multi-tenancy hardening is named skeleton work), rewritten into the
release manifest pre-apply and asserted against the running deployment; (4) runner
pods need the `tf-runner` ServiceAccount in the CR's own namespace — now compiled
into the stack namespace by the managed emitter (static, deterministic wiring);
(5) **upstream provider defect, open:** the destroy APPLY fails repeatably with
"unexpected end of JSON input" (empty-body JSON-decode class) though the destroy
plan succeeds — provider-version bisect in progress. **The managed scenario itself
PASSED live twice** (real branch provisioned through Flux + tofu-controller, SQL
over TLS using only the written-outputs secret at the declared secretRef name).
Two teardown leaks occurred while the destroy leg was broken; both caught by the
harness's API leak-check, both deleted (one manually by the steward, one by the
harness's new self-cleaning-but-still-failing remediation). Owner's project
verified clean after each. GPG remains locked; all commits held.

**Slice 5 LANDED (2026-07-26, commit `29e9364`; Revision 4 record `136c47d`).**
Defect (5) resolved — not a provider version: a missing `depends_on` between
`neon_role` and `neon_endpoint` let OpenTofu parallelize create/destroy, racing
into the Neon SDK's unchecked empty-body JSON decode (real upstream fragility,
no matching issue filed; reachable only via the race). One dependency edge fixed
both legs on the already-pinned 0.14.0, proven by toggling it against the real
API twice each way. Final state: managed scenario green end to end incl. clean
in-cluster destroy + API-verified no leak; self-hosted regression green; fast
tier + flake check green. Contract review dispatched post-landing.

## WEEK TWO COMPLETE; dogfood note 2 recorded (2026-07-26)

All five exit criteria of `docs/plans/02-week-two.md` checked off, each verified
by running. Owner decisions at landing (verbatim in the plan + problem
definition): Q1.1a team name STRUCK ("immaterial to this project" — denominator 5
stands); dogfood note 2 = the managed case (reversal of the separate-real-request
answer, flagged). Note 2: `dogfood/managed-api`, **time-to-stack 2m53s**, real
Neon branch via the full compiled path, probe from the written-outputs secret
only; stack persists (d7s-dogfood + Neon project). Four operator-friction
findings recorded in `docs/dogfood.md` (hardcoded component name in managed
actions; GitRepository registration not an operator affordance; hand-supplied
projectId; PodSecurity warning on the warm pod). **Kill review 2026-08-23: two
stacks recorded, caveats in the dogfood record.** Remaining owner item: push
main (first CI run of the full flake-gated pipeline; add NEON_API_KEY repo
secret for the managed tier).

## Week three begun (2026-07-26) — plan APPROVED Revision 1 (`8be0f82`)

Scope: durability triple whole via `external` object-store declaration (rpo
compiles labeled CONDITIONAL, Amendment 2's shape; "both, external first" — the
compiled component stays skeleton scope, owner-affirmed); live backup-completion
probe; emitted `healthChecks`; nix quick-start with built `d7s` in the dev shell;
note-2 operator affordances. Dispatched: slices 1+2 (external + durability
compile) and slice 5 (quick-start) in parallel on disjoint surfaces. Teammate
cold run (dogfood note 3) scheduled mid-week once quick-start lands — owner
picks the developer. Kill review 2026-08-23.

## Week-three slices 1+2+5 status (2026-07-26)

- **Slice 5 (quick-start) ACCEPTED**: `97c026d` + review fix `cc025d9`. `nix
  develop` provides a built `d7s`; QUICKSTART verified command-by-command (the
  implementer ran a full live acceptance to earn its claims); new CI quickstart
  job exercises the flake-built binary. One review finding (internal rule numbers
  leaking into an external doc) fixed. **Teammate cold run is unblocked.**
- **Slices 1+2 (external + durability CONDITIONAL) landed** as `caf04c5` — by a
  second implementer finishing the first's cancelled-but-sound partial work
  (owner stopped the original agent; fresh agent reviewed every hunk, kept the
  design, added the unwired CLI summary, rewrote stale tests, added the missing
  domain coverage). Schema change: `rpo` is now `{target, backupTo}`; refusals
  re-shaped with remedies naming the `external` block. All four test tiers green;
  contract review clean on code surfaces and notes the rule-9 watch item CLOSED
  by the compiler-core comment/structure. Three findings fixed and ACCEPTED
  (`921db90`): harness bad-rpo fixture reshaped to a real refusal (CI-safe —
  push unblocked), QUICKSTART rpo note now current truth, the claimed
  flux-level test now exists and passes. Slice 3 (live backup-completed probe
  vs harness MinIO; guarantee composition mtls+rpo on one component)
  dispatched.

## WEEK THREE BUILD COMPLETE — one exit criterion open (2026-07-26)

All six slices landed and accepted: quick-start (`97c026d`+`cc025d9`), external +
durability CONDITIONAL (`caf04c5`+`921db90` — completed by a second implementer
after the owner stopped the first mid-slice; partial work verified sound and
kept), live backup-completed probe (`1515e21`, zero review findings — the leg
failing by design since week one is green; both guarantee families composed on
one component), compiled healthChecks with harness compensation deleted
(`d5843bc`), operator-grade managed actions demonstrated live via `widgets-db`
(`2c7d8ae`+`d1ef150` — also live-caught and fixed a parallel-run image-tag
collision, and cleared the PodSecurity warning). Fast tier + flake check green on
merged HEAD; README refreshed (`0f8e140`, `90eb88f`). Five of six exit criteria
checked off, verified by running. **Open: dogfood note 3 — the teammate's cold
QUICKSTART run (owner picks the developer).** Kill review 2026-08-23.

## WEEK THREE COMPLETE — all six exit criteria checked (2026-07-26)

Dogfood note 3 recorded: the first human cold run (a real teammate) found the
piecemeal-path drift within minutes — the exact class of evidence no agent or CI
run could produce, since both always drive the orchestrator. Fixed and
re-verified same day (`12672d1`): documented commands run literally on a fresh
cluster; deliver's new prerequisite checks shown failing-with-remedy and
passing. Follow-on finding named for next planning: the self-hosted piecemeal
actions hardcode the example's names (week-one) — a novel self-hosted stack
can't be delivered piecemeal; the managed actions' parameterization pattern is
the template. Owner items: **push** (20 commits), thank the developer.
Kill review 2026-08-23: two real stacks, three dogfood notes, both guarantee
families live (one CONDITIONAL per contract), both placements proven.

## Note-3 fix rounds fully closed (2026-07-26)

The cold-run session's second finding (deliver silently waiting on an absent git
source) fixed in `0aed269`: gitserver prerequisite guard in both deliver actions,
and every bounded wait in the harness now announces itself at start. Review pass
on both fix commits: clean except one rule-39 finding (live-caught fixes without
an automated repro) — closed by `319ef67`: a cluster-free stub-kubectl
conformance test pinning all four prerequisite refusals (remedy strings included)
and the wait announcement, fail-then-pass proven by independently breaking both
code paths, wired into scripts/check.sh so CI's fast tier guards them. Week
three stands fully closed: six of six exit criteria, all findings fixed or
named. Owner: push (4 local commits).

## Week four begun (2026-07-26) — plan APPROVED Revision 1 (`a6b29a6`)

Ground: two owner directives verbatim in the plan (quickstart per-scenario
isolation + manual route; THE MESH IS MANDATORY — external endpoints reached
only through the mesh with permission). Flagged finding: the weekly plans'
egress deferral contradicted Amendment 2's "compiled default-deny; v1 = declare
+ deny" — contract wins, deferral superseded. Approved answers: declared-
endpoints-first deny breadth (namespace-wide deny-all is the named next step);
manual route in companion WALKTHROUGH.md. Build order: slices 1+2 (egress
compiler + allowedConsumers un-refusal on managed — dispatched), then 3
(mesh-mandatory scenarios + negative egress probes), then 4 (docs).

## Week-four slices 1+2 landed and accepted; slice 3 dispatched (2026-07-26)

`29abd8c` (egress compiler): waypoint-per-namespace + ServiceEntry +
identity-scoped AuthorizationPolicy from declared wiring only; the ambient-label
gap fixed (namespace joins the mesh whenever egress is needed, not only for
mtls); allowedConsumers un-refused on managed; *.neon.tech DYNAMIC_DNS wildcard
disclosed as a precision limit with the SNI question flagged for live proof.
Review: clean incl. the inert-policy check (use-waypoint bound); one minor
rule-33 aggregation finding returned as a fix commit (in flight). Slice 3
dispatched in parallel: mesh-mandatory scenarios, positive probes through the
mesh, negative undeclared-workload egress probes both scenarios, the SNI
empirical answer. Both full live runs required before its commit.

## Live-caught composition bug at the trust boundary (2026-07-26/27)

Slice 3's managed run: the compiled egress enforcement BLOCKED d7s's own
provisioner — tf-runner's `terraform plan` EOF'd calling console.neon.tech:443,
which matches the *.neon.tech ServiceEntry but is allowed to no identity and no
port besides consumers:5432. Week-one's rule-42 composition class recurring at
the boundary, caught by the deny-by-default posture exactly as designed. Fix
authorized within plan slice 1 (the provisioner edge IS declared wiring implied
by placement: managed): emit the control-plane edge — 443 scoped to the
d7s-compiled tf-runner ServiceAccount only; consumers keep 5432 only. Emitter
fix in flight; slice 3 re-runs managed live after. Self-hosted half already
green: backup through the waypoint, undeclared identity refused with
diagnostics.

## Design finding: wildcard egress unprogrammed by ambient waypoints (2026-07-27)

Steward-run experiment on the debug cluster (implementer stalled; two-command
diagnosis run directly): legacy Postgres negotiation fails (SSLRequest bytes,
waypoint kills), direct TLS (sslnegotiation=direct, SNI-first) ALSO fails —
ztunnel shows probe traffic reaching the waypoint and closed in 1ms both modes;
istioctl proxy-config proves the waypoint has cluster+route for the exact-host
console.neon.tech:443 SE and NOTHING for the wildcard *.neon.tech:5432
DYNAMIC_DNS SE. Istio 1.30.3 does not program wildcard SEs into waypoints. The
compiled data-plane edge fails CLOSED (denied for everyone — safe, not correct:
declared consumers should pass). Control-plane edge (443) fully proven live.
Upstream research dispatched (support matrix, blessed ambient egress patterns
for dynamic hosts); design decision round to follow with facts. Self-hosted
egress proof complete and green.

## DAY CLOSE — 2026-07-27 (session end; next session resumes here)

Week-four state at close: slices 1+2 landed through THREE evidence-driven
revisions (wildcard → alpha-gated upstream → exact-host pinning; halfway deny →
REGISTRY_ONLY dead end → COMPILED NetworkPolicy floor, `b3336a9`). Proven live:
self-hosted egress both probes; managed ceremony end to end (two-phase pin);
authorization-deny at registered endpoints. NOT yet proven: the NetworkPolicy
floor itself (landed compiled, never live-run) and its disclosed kube-apiserver
gap — kind v0.31 enforces NetworkPolicy natively, so **the next CI run on this
push IS the scheduled experiment**: if CNPG/tf-runner cannot reach the API
server under the compiled floor, acceptance goes red and that evidence drives
the next decision (schema knob vs CNI-exemption reliance). UNCOMMITTED in the
working tree (staged, deliberately held): the harness implementer's slice-3 WIP
— two-phase ceremony actions, mesh-mandatory managed scenario, both new probe
legs, and an istio-install REGISTRY_ONLY change that MUST BE REVERTED before
landing (dead end; see the plan's mechanism correction). Next session's opening
moves: (1) read CI's verdict on this push; (2) resume the harness implementer's
held work — revert REGISTRY_ONLY, deliver the compiled floor, prove both deny
legs live in both scenarios; (3) contract reviews; (4) slice 4 docs
(QUICKSTART isolation + WALKTHROUGH.md, incl. the ceremony, the istiod-HPA
note, and CNI prerequisites); (5) week-four exit criteria. Kill review
2026-08-23; two stacks + three dogfood notes on record.

## Open items owned by the owner

- **Q1.1a**: RESOLVED 2026-07-26 — denominator (5) recorded; team name struck by
  owner decision (verbatim in the problem definition).
- **Week-two plan approval**: RESOLVED 2026-07-26 — **APPROVED as Revision 1** with
  one owner change (verbatim in the plan's "Approval round" section): Neon is
  bootstrapped locally, simulated control-plane surface named where needed, on
  12-factor dev/prod-parity grounds; conflict with the as-drafted exit criterion
  flagged and resolved in the plan, not silently. Teardown: harness ephemeral,
  dogfood persists. Dogfood note 2 comes from a separate real request. Build starts
  at slice 1 (now a feasibility spike: tofu-controller health + local-Neon path).
- **Push main**: 12+ local commits unpushed; first real run of the flake-wired CI
  acceptance job waits on it.
