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

## Open items owned by the owner

- **Q1.1a**: denominator (5) recorded 2026-07-26; the **team name** is still
  unrecorded — the open sliver, flagged in the problem definition.
- **Week-two plan approval**: RESOLVED 2026-07-26 — **APPROVED as Revision 1** with
  one owner change (verbatim in the plan's "Approval round" section): Neon is
  bootstrapped locally, simulated control-plane surface named where needed, on
  12-factor dev/prod-parity grounds; conflict with the as-drafted exit criterion
  flagged and resolved in the plan, not silently. Teardown: harness ephemeral,
  dogfood persists. Dogfood note 2 comes from a separate real request. Build starts
  at slice 1 (now a feasibility spike: tofu-controller health + local-Neon path).
- **Push main**: 12+ local commits unpushed; first real run of the flake-wired CI
  acceptance job waits on it.
