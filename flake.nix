{
  description = "codesieve - token-efficient local code retrieval CLI for coding agents";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = "0.1.0-dev";
        codesieve = pkgs.buildGoModule {
          pname = "codesieve";
          inherit version;
          src = ./.;
          vendorHash = "sha256-MR943XkJosza5+BSSA6cwqf/25zxPLiG84ARz0xkSyg=";
          subPackages = [ "cmd/codesieve" ];

          ldflags = [ "-s" "-w" "-X main.version=${version}" ];

          checkPhase = ''
            runHook preCheck
            go test ./...
            runHook postCheck
          '';
        };
      in {
        packages.default = codesieve;

        apps.default = {
          type = "app";
          program = "${codesieve}/bin/codesieve";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            bats
            jq
            sqlite
            clang
          ];
        };

        checks = {
          build = codesieve;
          tests = pkgs.runCommand "codesieve-tests" {
            nativeBuildInputs = [ pkgs.bats pkgs.jq ];
          } ''
            cp -R ${self} source
            chmod -R +w source
            cd source
            export HOME=$TMPDIR
            export CODESIEVE_BIN=${codesieve}/bin/codesieve
            bats tests/bats
            touch $out
          '';
        };
      });
}
