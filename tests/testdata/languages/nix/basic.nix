{ pkgs, system, ... }:
let
  mkApp = name: {
    pname = name;
  };
in {
  packages.${system}.default = pkgs.hello;
  devShells.${system}.default = pkgs.mkShell {
    buildInputs = [ pkgs.go ];
  };
  overlays.default = final: prev: {
    hello = prev.hello;
  };
  apps.${system}.default = mkApp "codesieve";
}
