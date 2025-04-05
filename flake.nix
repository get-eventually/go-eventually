{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };

          go = pkgs.go_1_24;
          withOurGoVersion = pkg: pkg.override { buildGoModule = pkgs.buildGo124Module; };

          gopls = withOurGoVersion pkgs.gopls;
          delve = withOurGoVersion pkgs.delve;
        in
        {
          devShells.default = with pkgs; mkShell {
            packages = [
              go
              buf
            ] ++ [
              gopls
              delve
              goreleaser
            ] ++ (map withOurGoVersion [
              gotools
              go-outline
              gopkgs
            ]) ++ [
              git
              nil
              golangci-lint
              markdownlint-cli
            ];

            # Provide binary paths for tooling through environment variables.
            GO_BIN_PATH = "${go}/bin/go";
            GOPLS_PATH = "${gopls}/bin/gopls";
            DLV_PATH = "${delve}/bin/dlv";
          };
        }
      );
}
