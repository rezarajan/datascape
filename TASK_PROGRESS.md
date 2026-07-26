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

## Open items owned by the owner

- **Q1.1a**: team name + developer-customer count — record (privately if repo goes
  public) before the first dogfood week, so adoption claims have a denominator. Still
  open as of 2026-07-26.
- **Managed-emit artifact** (deferred at Revision A approval, blocks week two only):
  Flux tf-controller `Terraform` CR (recommended) vs plain Terraform module vs
  Crossplane claim.
