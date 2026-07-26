{
  # Every binary the checks and the acceptance harness invoke
  # (scripts/acceptance-kind.sh) comes from this dev shell, so local runs
  # and CI exercise the same pinned toolchain: `nix develop --command
  # scripts/acceptance-kind.sh`. The Docker daemon kind talks to is a host
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
    };
}
