{
  inputs.nixpkgs.url = "nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    let
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
      version = (builtins.substring 0 8 lastModifiedDate) + "-" + (if (self ? rev) then self.rev else "dirty");
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
      libFor = forAllSystems (system: import (nixpkgs + "/lib"));
      nixosLibFor = forAllSystems (system: import (nixpkgs + "/nixos/lib"));
    in flake-utils.lib.eachSystem supportedSystems (system: let 
      pkgs = import nixpkgs {
        inherit system;
      };
      lib = import (nixpkgs + "/lib") {
        inherit system;
      };
      nixosLib = import (nixpkgs + "/nixos/lib") {
        inherit system;
      };
    in rec {
      devShells = let pkgs = nixpkgsFor.${system}; in { default = pkgs.mkShell {
          buildInputs = with pkgs; [
            cargo
            gnumake
          ];
      }; };
  });
}
