{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.gotools
    pkgs.pre-commit
    pkgs.ninja
    pkgs.nix-bundle
  ];
}
