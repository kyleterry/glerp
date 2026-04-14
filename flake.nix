{
  description = "glerp: embeddable Scheme interpreter in Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells = with pkgs; let
          common = [
            go_1_26
            go-task
            golangci-lint
          ];
        in {
          default = mkShell {
            packages = common ++ [
              gotools
              golines
              gopls
            ];
          };

          ci = mkShell {
            packages = common;
          };
        };
      });
}
