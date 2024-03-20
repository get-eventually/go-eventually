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

          # NOTE: fixing Go 1.21 for the project.
          go = pkgs.go_1_21;
        in
        {
          devShells.default = with pkgs; mkShell {
            nativeBuildInputs = [
              go
              buf
            ];

            packages = [
              gopls
              gotools
              go-outline
              gopkgs
              delve

              # Linters
              nil
              golangci-lint
              markdownlint-cli
            ];

            # Provide binary paths for tooling through environment variables.
            GO_BIN_PATH = "${go}/bin/go";
            GOPLS_PATH = "${gopls}/bin/gopls";
            DLV_PATH = "${delve}/bin/dlv";

            # Disable ryuk container for testcontainers.
            TESTCONTAINERS_RYUK_DISABLED = "true";
          };
        }
      );
}
