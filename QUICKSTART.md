# Quickstart

For a developer with [Nix](https://nixos.org/download) and
[Docker](https://docs.docker.com/get-docker/) already installed. Every command
below was actually run to write this doc — nothing here is aspirational.

## 1. Clone and enter the dev shell

```
git clone <this repo> && cd datascape
nix develop
```

`nix develop` builds `d7s` from source the first time (cached after) and drops you
in a shell with the compiled binary already on `PATH` — no separate `go build`
step. `go` is on `PATH` too, since you'll still want it to run tests or read the
source. Confirm it's there:

```
d7s
# d7s: usage: d7s compile <file> -o <dir>
```

(`d7s` has no `--help` yet — the bare invocation above is its usage message. Trying
an unknown subcommand refuses the same way: "planned, not yet available".)

You can also run the compiled binary without entering the shell at all:
`nix run .#d7s -- compile ...`.

## 2. Declare a stack

d7s compiles a **Stack** declaration — the smallest valid one, one Postgres
component with the transport-security guarantee on:

```yaml
apiVersion: d7s.dev/v1alpha1
kind: Stack
name: quickstart
components:
  - kind: postgres
    name: orders-db
    placement: self-hosted        # or "managed" — see the scope note below
    credentials:
      secretRef:
        name: orders-db-app       # a Kubernetes Secret name — d7s never accepts secret values
    guarantees:
      mtls: {}                    # turns on mesh mTLS; once declared it can't be turned off
    allowedConsumers:
      - serviceAccount: probe-client   # who the mTLS allow-list lets in
```

Save that as `stack.yaml`. Each field:

- `name` (stack-level) — the stack's own name; shows up in emitted paths.
- `components[].kind` — the component type; `postgres` is the only kind today.
- `components[].name` — the component's name (also its Kubernetes resource name).
- `placement` — where it runs: `self-hosted` (CloudNativePG on your cluster) or
  `managed` (Neon — see the scope note below).
- `credentials.secretRef.name` — the name of a Secret you (or the operator)
  create ahead of time; d7s never accepts or emits a credential value.
- `guarantees.mtls` — declares the transport-security guarantee; presence alone
  turns it on, self-hosted placement only.
- `allowedConsumers[].serviceAccount` — the identities the compiled
  `AuthorizationPolicy` allow-lists; everything else is denied by default.

## 3. Compile it

```
d7s compile stack.yaml -o ./out
```

This is the only thing `d7s` does — read-only, deterministic, no cluster contact
(compiling twice produces byte-identical output). `./out` gets:

```
out/flux/                 # the two Flux Kustomizations Flux itself reconciles
out/infra/cnpg-operator/  # the CNPG operator install (HelmRepository + HelmRelease)
out/apps/<stack-name>/    # Namespace, Cluster CR, PeerAuthentication, AuthorizationPolicy
```

d7s never applies any of this — that's Flux's job, next.

## 4. Deliver it

Two ways to see the compiled output actually running, on a throwaway
[kind](https://kind.sigs.k8s.io/) cluster:

- **Watch the whole documented scenario run itself:**
  ```
  nix run .#acceptance
  ```
  Stands up kind, installs Flux + Istio ambient + CNPG, pushes `out/` to a git
  source, lets Flux reconcile it, runs the live mTLS probes, tears everything
  down. This is the same command CI runs on every push.

- **Piecemeal**, one step at a time (each is its own flake action, useful if you
  want to inspect state between steps): `nix run .#cluster-up`, `.#flux-install`,
  `.#istio-install`, `.#minio-install`, `.#git-source`, `.#deliver`, `.#guard`,
  `.#probes`, `.#durability-probe`, `.#teardown`. See `scripts/actions/` for
  what each one does — this is the same order `nix run .#acceptance` runs them
  in (`scripts/actions/acceptance.sh`); `minio-install` stands up the declared
  `external` object store the durability guarantee backs up to, and must run
  before `deliver` materializes its app-namespace credentials secret.

### The managed scenario (`placement: managed`)

The same shape, delivered to a real Neon database instead of CNPG: `nix run
.#acceptance-managed` runs it end to end (its own throwaway kind cluster, so it
never collides with the self-hosted one above), or piecemeal: `.#compile-managed`,
`.#cluster-up`, `.#flux-install`, `.#tofu-install`, `.#git-source`,
`.#deliver-managed`, `.#probe-managed`, `.#teardown-managed`. `deliver-managed`
registers the `GitRepository` for you — but that's the same one-per-cluster
registration the self-hosted flow needs (see the prerequisite below); if you're
driving the managed scenario by hand instead of through these actions, you still
need it registered, pointing at wherever you pushed the compiled `out/`, before
Flux can reconcile the Terraform CR at all. If your component isn't named
`orders-db` (the example's name, and the default), tell the actions about yours
via environment variables: `MANAGED_STACK` (your declaration file),
`MANAGED_NAMESPACE` (your stack name), `MANAGED_COMPONENT` (your component name),
and `MANAGED_CREDENTIALS_SECRET` (your declared credentials secret) — the actions
read these, not your YAML.

### Environment prerequisites (not compiled by d7s — you provide these)

- **Flux** installed on the target cluster (`nix run .#flux-install`, or your own).
- **Istio ambient mode** installed, for `guarantees.mtls` to have a mesh to enforce
  against (`nix run .#istio-install`).
- **The credentials Secret already exists** before applying the Cluster CR — CNPG's
  `bootstrap.initdb.secret` only consumes a pre-existing secret, it doesn't create
  one.
- **A `GitRepository` registered with Flux**, pointing at wherever you pushed
  `out/` — the emitted Kustomizations name it as their `sourceRef`, they don't
  create it themselves (`nix run .#git-source` stands up the harness's own
  in-cluster git source and pushes `out/` to it; `deliver`/`deliver-managed`
  register the `GitRepository` against it and refuse with a remedy if it's
  missing, rather than leaving Flux to DNS-fail silently through a bounded
  wait).
- **A `StorageClass` with `reclaimPolicy: Retain`**, if you need data volumes
  retained after a component is removed from the declaration — this is purely a
  cluster-provided `StorageClass` property; d7s can't safely guess your CSI
  provisioner.

## Honest scope notes

- **`placement: managed`** compiles to Neon via tofu-controller and needs a real
  `NEON_API_KEY` (env var or a gitignored `./.env`) to deliver — there's no
  managed-placement path without one. It also needs to know which Neon project
  that key is scoped to: by default this is auto-discovered for you (Neon's API
  doesn't expose a direct "whose key is this" lookup, so the harness infers it
  from a deliberate, harmless API call), which costs one extra network round
  trip and assumes the key is scoped to exactly one project. Set `NEON_PROJECT_ID`
  yourself — env var or `./.env`, same as the API key — to skip discovery
  entirely or to pick a specific project when that assumption doesn't hold.
- **`guarantees.rpo` compiles once it names a destination.** Declare an `external`
  object store (v1's only shape: an S3-compatible endpoint/bucket, credentials as a
  Secret reference — never inline) and point `guarantees.rpo.backupTo` at its name:

  ```yaml
  external:
    - name: backups
      objectStore:
        endpoint: http://minio.d7s-harness-minio.svc:9000
        bucket: d7s-backups
        credentials:
          secretRef:
            name: backups-credentials
  components:
    - kind: postgres
      name: orders-db
      placement: self-hosted
      guarantees:
        rpo:
          target: 1h
          backupTo: backups
  ```

  d7s never provisions or reaches into that store — it's outside what d7s compiled,
  so the compiled Cluster and ScheduledBackup carry a `conditional-on-external`
  label, and the compile summary prints the same notice: the durability claim is
  only as strong as that external store's own guarantees. A bare `guarantees.rpo`
  with no `backupTo`, or a `backupTo` naming an external you didn't declare, still
  refuses to compile with the remedy in the error. `placement: managed` still
  refuses `guarantees.rpo` outright — there's no destination wiring for it yet.
