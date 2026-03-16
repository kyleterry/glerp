{
  description = "glerp — embeddable Scheme interpreter in Go";

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
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go_1_26
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.golines
            pkgs.gopls
          ];
        };
      });
}
