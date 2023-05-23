{
  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/nixos-22.11;
    flake-utils.url = github:numtide/flake-utils;
  };

  outputs = { self, nixpkgs, flake-utils, poetry2nix, ... }@attrs: flake-utils.lib.eachSystem [ "x86_64-linux" ] (system: let
    inherit (poetry2nix.legacyPackages.${system}) mkPoetryApplication mkPoetryEnv;
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.default = pkgs.mkShell {
      buildInputs = with pkgs; [
        arduino-cli
        picocom
        clang
      ];
    };
  });
}
