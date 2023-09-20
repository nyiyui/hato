{
  inputs.nixpkgs.url = github:NixOS/nixpkgs/nixos-22.11;
  outputs = { self, nixpkgs }: let
    pkgs = nixpkgs.legacyPackages.x86_64-linux;
  in {
    devShells.x86_64-linux.default = pkgs.mkShell {
      buildInputs = with pkgs; [ arduino-cli ];
    };
  };
}
