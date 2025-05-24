{
  description = "Hugefiver's NixOS configure file";

  inputs = {
    # nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable-small";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.05-small";
    nixpkgs-unstable.url = "github:nixos/nixpkgs/nixpkgs-unstable";

    home-manager.url = "github:nix-community/home-manager";
    home-manager.inputs.nixpkgs.follows = "nixpkgs";

    disko.url = "github:nix-community/disko";
    disko.inputs.nixpkgs.follows = "nixpkgs";

    nixos-facter-modules.url = "github:numtide/nixos-facter-modules";
  };

  outputs = {
    self,
    nixpkgs,
    nixpkgs-unstable,
    home-manager,
    disko,
    ...
  } @ inputs: let
    mkPkgs = nixpkgs: system: import nixpkgs {inherit system;};
  in {
    nixosConfigurations = {
      nixos-txsh = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;
          pkgs = mkPkgs nixpkgs system;

          unstable = mkPkgs nixpkgs system;
        };

        modules = [
          ./hosts/nixos-txsh
        ];
      };

      nixos-txjp = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;

          pkgs = mkPkgs nixpkgs system;
          unstable = mkPkgs nixpkgs-unstable system;
        };

        modules = [
          ./hosts/nixos-txjp
        ];
      };

      bwh1 = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;

          pkgs = mkPkgs nixpkgs system;
          unstable = mkPkgs nixpkgs-unstable system;
        };

        modules = [
          ./hosts/bwh1
        ];
      };
    };
  };
}
