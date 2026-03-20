{
  description = "Hugefiver's NixOS configure file";

  inputs = {
    # nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable-small";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.11-small";
    nixpkgs-big.url = "github:nixos/nixpkgs/nixos-25.11";
    nixpkgs-unstable.url = "github:nixos/nixpkgs/nixpkgs-unstable";

    home-manager.url = "github:nix-community/home-manager";
    home-manager.inputs.nixpkgs.follows = "nixpkgs";

    disko.url = "github:nix-community/disko";
    disko.inputs.nixpkgs.follows = "nixpkgs";

    microvm.url = "github:microvm-nix/microvm.nix";
    microvm.inputs.nixpkgs.follows = "nixpkgs";

    nixos-facter-modules.url = "github:numtide/nixos-facter-modules";
  };

  outputs = {
    self,
    nixpkgs,
    nixpkgs-big,
    nixpkgs-unstable,
    home-manager,
    disko,
    microvm,
    ...
  } @ inputs: let
    mkPkgs = nixpkgs: system: import nixpkgs {inherit system;};
    default = {
      config,
      pkgs,
      lib,
      ...
    }: {
      # environment.systemPackages = lib.mkAfter (with pkgs; [
      #   nixos-rebuild-ng
      # ]);
    };
    defaultHm = {
      self,
      unstable,
      ...
    }: {
      imports = [
        home-manager.nixosModules.home-manager
      ];

      home-manager.extraSpecialArgs = {
        inherit self unstable;
      };
      home-manager.users.hugefiver = import ./hm/me.nix;
    };
  in {
    nixosConfigurations = {
      nixos-txsh = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;

          # pkgs = mkPkgs nixpkgs system;
          unstable = mkPkgs nixpkgs-unstable system;
        };

        modules = [
          # nixpkgs.nixosModules.readOnlyPkgs

          ./hosts/nixos-txsh
        ];
      };

      nixos-txjp = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;

          unstable = mkPkgs nixpkgs-unstable system;
        };

        modules = [
          # nixpkgs.nixosModules.readOnlyPkgs

          default
          ./hosts/nixos-txjp
        ];
      };

      bwh1 = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;

          unstable = mkPkgs nixpkgs-unstable system;
        };

        modules = [
          # nixpkgs.nixosModules.readOnlyPkgs

          default
          ./hosts/bwh1
        ];
      };

      nixos-ccus = nixpkgs.lib.nixosSystem rec {
        system = "x86_64-linux";

        specialArgs = {
          inherit self inputs system;

          unstable = mkPkgs nixpkgs-unstable system;
        };

        modules = [
          # nixpkgs.nixosModules.readOnlyPkgs

          default
          ./hosts/cc-us
        ];
      };
    } // (let
      system = "x86_64-linux";
      nixpkgs = nixpkgs-big;
      unstable = mkPkgs nixpkgs-unstable system;
      # inputs = {
      #   inherit self nixpkgs unstable home-manager disko;
      # };
      specialArgs = {
        inherit self inputs system unstable;
      };
     in {
      desktop-nuc13 = nixpkgs.lib.nixosSystem rec {
        inherit system specialArgs;
        modules = [
          ./desktops/nuc13
          defaultHm
        ];
      };

      desktop-nuc13-installer-vm = nixpkgs.lib.nixosSystem rec {
        inherit system specialArgs;
        modules = [
          ./tests/microvm/desktop-nuc13-installer.nix
        ];
      };
     });

    packages.x86_64-linux = {
      disko-install = inputs.disko.packages.x86_64-linux.disko-install;
      desktop-nuc13-installer-vm =
        self.nixosConfigurations.desktop-nuc13-installer-vm.config.microvm.declaredRunner;
    };
  };
}
