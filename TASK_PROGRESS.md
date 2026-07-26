# TASK_PROGRESS — solution setup + week-one plan

Resumability file per `docs/foundations/agentic-development.md` §4. Branch:
`claude/datascape-project-kickoff-ai83d9`. A session resuming this task needs only this
file plus `git log`.

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
   - (f) Acceptance harness on kind — **BLOCKED, not started.** See finding below.
     This is the exit-criteria checklist item (rule 58: verified by running, never
     from memory) — none of it is checked off yet, and none of the guarantee triples'
     "conformance probe" leg has been proven against a real cluster.

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

## Open items owned by the owner

- **Q1.1a**: team name + developer-customer count — record (privately if repo goes
  public) before the first dogfood week, so adoption claims have a denominator. Still
  open as of 2026-07-26.
- **Managed-emit artifact** (deferred at Revision A approval, blocks week two only):
  Flux tf-controller `Terraform` CR (recommended) vs plain Terraform module vs
  Crossplane claim.
