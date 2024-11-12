{
  description = "Hugefiver's NixOS configure file";

  inputs = {
    home-manager.url = "github:nix-community/home-manager";
    # nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable-small";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-24.05-small";
    nixpkgs-unstable.url = "github:nixos/nixpkgs/nixpkgs-unstable";

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
  } @ inputs: {
    nixosConfigurations ={
      nixos-txsh = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs nixpkgs nixpkgs-unstable system;
        };

        modules = [
          ./hosts/nixos-txsh
        ];
      };
    };
  };
}
