{
  self,
  inputs,
  lib,
  pkgs,
  unstable,
  system,
  ...
} : {
  imports =
    [
      # Include the results of the hardware scan.
      #./hardware-configuration.nix

      inputs.disko.nixosModules.disko
      inputs.nixos-facter-modules.nixosModules.facter

      (import ./desktop-common-pkgs.nix {pkgs = unstable;})

    ];

  # hardware.facter.reportPath = ./facter.json;

  nix.settings.substituters = lib.mkForce [
    "https://mirrors.cernet.edu.cn/nix-channels/store"
    "https://mirror.iscas.ac.cn/nix-channels/store"
    # "https://mirror.sjtu.edu.cn/nix-channels/store"
    # "https://mirrors.sjtug.sjtu.edu.cn/nix-channels/store"
    "https://cache.nixos.org"
  ];

  nix.settings.experimental-features = ["nix-command" "flakes"];

  boot.kernelPackages = lib.mkDefault pkgs.linuxPackages_latest;

  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  # networking.hostName = "hostname";
  networking.networkmanager.enable = true;

  time.timeZone = "Asia/Shanghai";
  i18n.defaultLocale = "en_US.UTF-8";

  boot.initrd.kernelModules = [ "bcache" ];
  # services.btrfs.swapfile.create = {
  #   size = "8G";
  # };

  environment.variables = {
    EDITOR = "vim";
  };

  environment.systemPackages = with unstable; [
    docker-compose
  ];

  services.logrotate.enable = true;
  services.journald.extraConfig = ''
    Compress=yes
  '';

  virtualisation.docker = {
    enable = true;
    storageDriver = "btrfs";
  };
  systemd.services.docker.unitConfig.Restart = "on-failure";

  users.extraGroups.docker.members = ["root" "hugefiver"];

  security.sudo = {
    enable = true;
    wheelNeedsPassword = false;
  };

  system.stateVersion = "25.11";
}