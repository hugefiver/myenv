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

    # 仅作为 hyprexpo 源码提供者：nixpkgs 中的 hyprlandPlugins.hyprexpo 版本
    # 常滞后于 hyprland，导致编译失败（如 hyprland 0.54.x + hyprexpo 0.53.0
    # 的 HookSystemManager.hpp 头文件路径变更）。我们只覆盖 src，hyprland
    # 主体仍使用 nixpkgs 缓存。
    hyprland-plugins.url = "github:hyprwm/hyprland-plugins";
    hyprland-plugins.flake = false;
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
          unstable = mkPkgs nixpkgs-unstable system {
            config.allowUnfree = true;
            overlays = [
              # 用 hyprland-plugins flake input 的源码覆盖 nixpkgs 中可能版本
              # 错配的 hyprexpo，hyprland 主体仍走 nixpkgs 二进制缓存。
              (final: prev: {
                hyprlandPlugins = prev.hyprlandPlugins // {
                  hyprexpo = prev.hyprlandPlugins.hyprexpo.overrideAttrs (_old: {
                    src = "${inputs.hyprland-plugins}/hyprexpo";
                    version = "git-${inputs.hyprland-plugins.shortRev or "dirty"}";
                  });
                };
              })
            ];
          };
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
