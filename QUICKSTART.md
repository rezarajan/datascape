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
  `.#istio-install`, `.#git-source`, `.#deliver`, `.#guard`, `.#probes`,
  `.#teardown`. See `scripts/actions/` for what each one does.

### Environment prerequisites (not compiled by d7s — you provide these)

- **Flux** installed on the target cluster (`nix run .#flux-install`, or your own).
- **Istio ambient mode** installed, for `guarantees.mtls` to have a mesh to enforce
  against (`nix run .#istio-install`).
- **The credentials Secret already exists** before applying the Cluster CR — CNPG's
  `bootstrap.initdb.secret` only consumes a pre-existing secret, it doesn't create
  one.
- **A `GitRepository` registered with Flux**, pointing at wherever you pushed
  `out/` — the emitted Kustomizations name it as their `sourceRef`, they don't
  create it themselves.
- **A `StorageClass` with `reclaimPolicy: Retain`**, if you need data volumes
  retained after a component is removed from the declaration — this is purely a
  cluster-provided `StorageClass` property; d7s can't safely guess your CSI
  provisioner.

## Honest scope notes

- **`placement: managed`** compiles to Neon via tofu-controller and needs a real
  `NEON_API_KEY` (env var or a gitignored `./.env`) to deliver — there's no
  managed-placement path without one. `NEON_PROJECT_ID` is auto-discovered from the
  key, or set it yourself to override.
- **`guarantees.rpo` currently refuses to compile, on every placement** — v1 has no
  declarable backup destination yet. It's landing this week (see
  `docs/plans/03-week-three.md`); until then, the compiler tells you as much and
  names the remedy.
