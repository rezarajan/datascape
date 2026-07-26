{
  # Every binary the checks and the acceptance harness invoke comes from
  # this flake, so local runs and CI exercise the same pinned toolchain.
  # The acceptance harness itself (docs/plans/02-week-two.md Revision 3,
  # slice 6) is decomposed into small, human-readable actions under
  # scripts/actions/ — each exposed as `nix run .#<action>`, shellcheck-
  # verified at build time with its own pinned runtimeInputs — composed
  # by the thin `acceptance` orchestrator: `nix run .#acceptance`. The
  # Docker daemon kind (and the git-source action) talk to is a host
  # prerequisite — a flake cannot provide a running daemon.
  description = "Datascape (d7s) dev + acceptance toolchain";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      forEachSystem = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      # Each action is scripts/lib/common.sh (shared config + helpers —
      # one source of truth for the bounded `poll` wait, golden rule 44)
      # embedded ahead of the unit's own body (scripts/actions/<name>.sh),
      # then built with writeShellApplication: shellcheck + `bash -n` at
      # build time, PATH pinned to exactly the listed runtimeInputs (plus
      # the ambient PATH, for host prerequisites like the Docker daemon's
      # own CLI where that's the deliberate exception).
      mkAction =
        pkgs: name: runtimeInputs: bodyFile:
        pkgs.writeShellApplication {
          inherit name runtimeInputs;
          text = ''
            ${builtins.readFile ./scripts/lib/common.sh}
            ${builtins.readFile bodyFile}
          '';
        };
    in
    {
      devShells = forEachSystem (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              kind
              kubectl
              fluxcd
              istioctl
              openssl
            ];
          };
        }
      );

      packages = forEachSystem (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          action = mkAction pkgs;
        in
        rec {
          compile-and-verify = action "compile-and-verify"
            (with pkgs; [
              go
              coreutils
              diffutils
              gnugrep
            ])
            ./scripts/actions/compile-and-verify.sh;

          cluster-up = action "cluster-up" (with pkgs; [ kind kubectl ]) ./scripts/actions/cluster-up.sh;

          flux-install = action "flux-install" [ pkgs.fluxcd ] ./scripts/actions/flux-install.sh;

          istio-install = action "istio-install" [ pkgs.istioctl ] ./scripts/actions/istio-install.sh;

          git-source = action "git-source"
            (with pkgs; [
              docker-client
              kind
              kubectl
              git
              coreutils
            ])
            ./scripts/actions/git-source.sh;

          deliver = action "deliver"
            (with pkgs; [
              kubectl
              fluxcd
              openssl
              coreutils
            ])
            ./scripts/actions/deliver.sh;

          guard = action "guard" [ pkgs.kubectl ] ./scripts/actions/guard.sh;

          probes = action "probes" (with pkgs; [ kubectl coreutils ]) ./scripts/actions/probes.sh;

          teardown = action "teardown" [ pkgs.kind ] ./scripts/actions/teardown.sh;

          # The thin orchestrator: same order, same trap-based ephemeral
          # teardown as the monolith it replaces. Each named unit is a
          # derivation exposing /bin/<name>, so listing them as
          # runtimeInputs puts every unit on the orchestrator's PATH.
          acceptance = action "acceptance"
            [
              compile-and-verify
              cluster-up
              flux-install
              istio-install
              git-source
              deliver
              guard
              probes
              teardown
            ]
            ./scripts/actions/acceptance.sh;

          # --- managed/Neon scenario (docs/plans/02-week-two.md
          # Revision 3, slice 5) — a separate composed entry point from
          # `acceptance` above, not chained after it: see
          # scripts/actions/acceptance-managed.sh for why.

          tofu-install = action "tofu-install"
            (with pkgs; [
              kubectl
              curl
              gnugrep
              gnused
              coreutils
            ])
            ./scripts/actions/tofu-install.sh;

          # curl/jq/gnugrep: discover_neon_project_id's Neon-API discovery
          # call (week-two plan Revision 4) - shared by neon-secret,
          # teardown-managed, and acceptance-managed below.
          neon-secret = action "neon-secret"
            (with pkgs; [
              kubectl
              curl
              jq
              gnugrep
            ])
            ./scripts/actions/neon-secret.sh;

          compile-managed = action "compile-managed" [ compile-and-verify ] ./scripts/actions/compile-managed.sh;

          deliver-managed = action "deliver-managed"
            (with pkgs; [
              kubectl
              fluxcd
              coreutils
              neon-secret
            ])
            ./scripts/actions/deliver-managed.sh;

          probe-managed = action "probe-managed" (with pkgs; [ kubectl coreutils ]) ./scripts/actions/probe-managed.sh;

          teardown-managed = action "teardown-managed"
            (with pkgs; [
              kubectl
              curl
              jq
              gnugrep
              teardown
            ])
            ./scripts/actions/teardown-managed.sh;

          acceptance-managed = action "acceptance-managed"
            (with pkgs; [
              curl
              jq
              gnugrep
              compile-managed
              cluster-up
              tofu-install
              flux-install
              git-source
              deliver-managed
              probe-managed
              teardown-managed
            ])
            ./scripts/actions/acceptance-managed.sh;
        }
      );

      apps = forEachSystem (
        system:
        nixpkgs.lib.mapAttrs (name: pkg: {
          type = "app";
          program = "${pkg}/bin/${name}";
        }) self.packages.${system}
      );
    };
}
