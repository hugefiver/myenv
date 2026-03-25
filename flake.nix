{
  description = "Hugefiver's NixOS configure file";

  inputs = {
    # nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable-small";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.11-small";
    nixpkgs-big.url = "github:nixos/nixpkgs/nixos-25.11";
    nixpkgs-unstable.url = "github:nixos/nixpkgs/nixpkgs-unstable";

    flake-parts.url = "github:hercules-ci/flake-parts";

    home-manager.url = "github:nix-community/home-manager";
    home-manager.inputs.nixpkgs.follows = "nixpkgs";

    disko.url = "github:nix-community/disko";
    disko.inputs.nixpkgs.follows = "nixpkgs";

    # microvm.url = "github:microvm-nix/microvm.nix";
    # microvm.inputs.nixpkgs.follows = "nixpkgs";

    nixos-facter-modules.url = "github:numtide/nixos-facter-modules";
  };

  outputs = inputs@{
    self,
    flake-parts,
    nixpkgs,
    nixpkgs-big,
    nixpkgs-unstable,
    home-manager,
    ...
  }:
    let
      supportedSystems = [
        "x86_64-linux"
      ];
      defaultSystem = builtins.head supportedSystems;
    in
      flake-parts.lib.mkFlake {inherit inputs;} ({...}: let
        inherit (nixpkgs) lib;

        mkPkgs = nixpkgsInput: system: extraAttrs:
          import nixpkgsInput (
            {
              inherit system;
            }
            // extraAttrs
          );

        mkSpecialArgs = system: {
          inherit self inputs system;
          unstable = mkPkgs nixpkgs-unstable system {};
        };

        commonNixosModule = {
          ...
        }: {
          # environment.systemPackages = lib.mkAfter (with pkgs; [
          #   nixos-rebuild-ng
          # ]);
        };

        mkNixos = {
          modules,
          nixpkgsInput ? nixpkgs,
          system ? defaultSystem,
        }:
          nixpkgsInput.lib.nixosSystem {
            inherit modules system;
            specialArgs = mkSpecialArgs system;
          };

        mkHome = system:
          home-manager.lib.homeManagerConfiguration {
            pkgs = mkPkgs nixpkgs-big system {
              config.allowUnfree = true;
              overlays = [
                (final: prev: {
                  xrdb = prev.xorg.xrdb;
                })
              ];
            };

            extraSpecialArgs = mkSpecialArgs system;

            modules = [
              ./hm/me.nix
            ];
          };

        nixosHosts = {
          nixos-txsh = {
            modules = [
              ./hosts/nixos-txsh
            ];
          };

          nixos-txjp = {
            modules = [
              commonNixosModule
              ./hosts/nixos-txjp
            ];
          };

          bwh1 = {
            modules = [
              commonNixosModule
              ./hosts/bwh1
            ];
          };

          nixos-ccus = {
            modules = [
              commonNixosModule
              ./hosts/cc-us
            ];
          };

          desktop-nuc13 = {
            nixpkgsInput = nixpkgs-big;
            modules = [
              ./desktops/nuc13
            ];
          };
        };

        desktopHome = mkHome defaultSystem;
      in {
        systems = supportedSystems;

        flake = {
          nixosConfigurations = lib.mapAttrs (_: host: mkNixos host) nixosHosts;

          homeConfigurations = {
            hugefiver = desktopHome;
            "hugefiver@desktop-nuc13" = desktopHome;
          };
        };
      });
}
