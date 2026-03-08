{
  description = "spank - Yells 'ow!' when you slap the laptop";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      # Home Manager module available regardless of system
      hmModule = import ./nix/hm-module.nix self;
    in
    flake-utils.lib.eachSystem [ "aarch64-darwin" "x86_64-linux" "aarch64-linux" ] (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = nixpkgs.lib;
        isDarwin = pkgs.stdenv.isDarwin;
      in
      {
        packages.default = pkgs.callPackage ./nix/package.nix { };

        devShells.default = pkgs.mkShell {
          inputsFrom = [ self.packages.${system}.default ];
          buildInputs = lib.optionals (!isDarwin) [ pkgs.alsa-utils ];
        };
      }
    ) // {
      homeManagerModules.default = hmModule;
      homeManagerModule = hmModule;
    };
}
