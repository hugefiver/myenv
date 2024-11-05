{
  description = "Hugefiver's NixOS configure file";

  inputs = {
    # nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    # use Tsinghua mirror
    nixpkgs.url = "git+https://mirrors.tuna.tsinghua.edu.cn/git/nixpkgs.git?ref=nixpkgs-unstable";
    # nixpkgs.url = "git+https://mirrors.tuna.tsinghua.edu.cn/git/nixpkgs.git?ref=nixos-unstable-small";

    home-manager.url = "github:nix-community/home-manager";
  };

  outputs = {
    self,
    nixpkgs,
    home-manager,
  }: let
    common = nixpkgs.lib.nixosSystem {
      modules = [
        ./hardware-config/hardware-config.nix
      ];
    };
  in {
    imports = [
      ./nix/common/common-pkgs.nix
    ];

    nixosConfigurations = rec {
      nixos = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";

        services.openssh = {
          port = 22;
        };
      };
    };
  };
}
