{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.pre-commit
    pkgs.ninja
    pkgs.nix-bundle
  ];
}
