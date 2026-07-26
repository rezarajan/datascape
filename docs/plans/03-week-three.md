# Week-three plan — the durability triple made whole, and the first human operator

**Class: Plan.** **Status: DRAFT (Revision 0), 2026-07-26 — awaiting owner approval.
No product code lands from this plan until approved (phase gate).**

Two inputs, both dated 2026-07-26: the steward's ranked proposal after week two
(complete the durability triple; fold in the two evidence-backed debt items), and the
owner's directive at this plan's request, recorded verbatim: *"for the teammate task
please ensure documentation is available for a quick-start. Working with the nix
route - it would be great if the developer could just pop into a dev environment
with the compiled binary available."*

## Why this week (anchor tests)

The contract (Amendment 1) names durability/recovery a v1 guarantee family; today
`guarantees.rpo` fails closed and exactly one family ships live — thin ground for a
guarantees compiler's kill review (2026-08-23). The rpo refusal was recorded as
"until a destination is declarable" — this week declares it. Evidence: the probe's
inability to pass has been documented since week one; slice 4 recorded the missing
`healthChecks`; dogfood notes 1–2 recorded the operator-affordance gaps and "no
installed binary." Nothing here is new breadth: every slice completes or hardens an
existing primitive.

## Slices

1. **`external` object-store declaration (Amendment 2's shape, first use).** The
   declaration model gains d7s's `source()`-style `external` block: an
   S3-compatible object store (endpoint, bucket, credentials `secretRef` — never
   values, rule 51) that d7s **never provisions or mutates** — outside the trust
   wall by definition. Unknown fields refuse; the declaration alone emits nothing.
2. **The durability triple compiles again, labeled CONDITIONAL.** `guarantees.rpo`
   + a declared external destination compiles: `barmanObjectStore` wired from the
   external declaration, ScheduledBackup from the RPO (the gated week-one machinery
   returns to service). Per Amendment 2, a durability guarantee crossing the wall
   compiles **labeled CONDITIONAL** — the label lands visibly in the compiled
   output (annotation `d7s.dev/guarantee-durability: conditional-on-external`) and
   in the CLI's compile summary; it is not a silent pass (rules 34/37 spirit: no
   unlabeled tier). `rpo` without a declared destination still refuses, with the
   remedy now naming the `external` block. Egress default-deny enforcement remains
   skeleton scope, named as deferred.
3. **Conformance probe: a backup completes.** The acceptance harness declares an
   external store (MinIO stood up by the harness as the environment's external
   thing — explicitly NOT d7s-compiled), and the probe verifies a `Backup` object
   reaches completed phase — the leg that has failed by design since week one goes
   green. Negative probes: rpo-without-destination refusal; the CONDITIONAL label
   present in output (the label can fail).
4. **`healthChecks` on emitted Kustomizations** (slice-4 live finding): the app
   Kustomization gains health checks on the operator deployment it depends on, so
   compiled `dependsOn` ordering holds without the harness's procedural compensation
   — which is then removed from the harness (the artifact earns what the script was
   doing).
5. **Quick-start + the compiled binary in the dev environment (owner directive).**
   The flake gains a `d7s` package (`buildGoModule`) exposed in the default
   devShell and as `nix run .#d7s` — `nix develop` drops a developer in front of a
   built binary (closes dogfood note 1's "no installed binary" finding). A
   `QUICKSTART.md` at the repo root (unclassified, like README) walks the cold
   path: clone → `nix develop` → declare → `d7s compile` → deliver on kind via the
   documented flow. Every command in it is exercised by CI or the harness
   (rule 41 — docs example = e2e test = demo).
6. **Operator affordances from dogfood note 2:** the managed actions parameterize
   the component name (no more hardcoded `orders-db`); the GitRepository
   environment prerequisite is documented where the operator meets it (QUICKSTART +
   plan prerequisites); the `tf-runner-warm` pod satisfies the restricted
   PodSecurity profile; the projectId affordance is documented (env var override
   exists).

## Explicitly NOT this week (deferred with a home)

Egress default-deny compilation and boundary probes (skeleton, Amendment 2); the
import ceremony (v2+); object storage as a d7s-COMPILED component (skeleton Q3.1 —
this week's store is external/declared only); attestation/declared=running; any new
component kind or second GitOps target; restore operation (day-2, refused).

## Exit criteria (verified by running — rule 58)

- [ ] A stack declaring `rpo` + an external store compiles deterministically;
      golden files pin the wired `barmanObjectStore`, ScheduledBackup, and the
      CONDITIONAL label; `rpo` without a destination still refuses with the
      new remedy (fail-then-pass tested).
- [ ] Live on kind: a real backup reaches completed phase against the
      harness-provided external MinIO; the probe that has failed since week one
      passes; teardown leaves nothing.
- [ ] Emitted `healthChecks` proven live: the harness's procedural operator-wait
      compensation is deleted and the scenario still passes on compiled ordering
      alone.
- [ ] `nix develop` provides a built `d7s`; QUICKSTART's commands run verbatim in
      CI/harness.
- [ ] **Dogfood note 3: a human teammate (not the owner, not an agent) runs the
      QUICKSTART cold and their outcome — success or friction — is recorded
      verbatim.** The kill review reads this as the first non-agent evidence.
- [ ] Managed actions run against a differently-named component (the note-2
      hardcoding finding closed, demonstrated).

## Open questions for the owner (at approval)

- **Durability destination shape** (blocks slices 1–3): external declaration with
  the CONDITIONAL label, as drafted (recommended — it is Amendment 2's own v1
  shape) vs compiling a MinIO component inside the wall (unconditional triple, but
  a whole new component kind — skeleton scope pulled forward).
- **Teammate + timing**: who runs the cold QUICKSTART, and when in the week —
  early enough that their friction can land as fixes before the kill review.
