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

  boot.loader.systemd-boot.enable = false;
  boot.loader.grub = {
    enable = true;
    efiSupport = true;
    device = "nodev";
    configurationLimit = 5;
  };
  boot.loader.efi.canTouchEfiVariables = true;
  # networking.hostName = "hostname";
  networking.networkmanager.enable = true;

  time.timeZone = "Asia/Shanghai";
  i18n.defaultLocale = "zh_CN.UTF-8";
  i18n.extraLocaleSettings = {
    LC_ADDRESS = "zh_CN.UTF-8";
    LC_IDENTIFICATION = "zh_CN.UTF-8";
    LC_MEASUREMENT = "zh_CN.UTF-8";
    LC_MONETARY = "zh_CN.UTF-8";
    LC_NAME = "zh_CN.UTF-8";
    LC_NUMERIC = "zh_CN.UTF-8";
    LC_PAPER = "zh_CN.UTF-8";
    LC_TELEPHONE = "zh_CN.UTF-8";
    LC_TIME = "zh_CN.UTF-8";
    LC_COLLATE = "en_US.UTF-8";
    LC_MESSAGES = "en_US.UTF-8";
  };
  i18n.supportedLocales = [ "zh_CN.UTF-8/UTF-8" "en_US.UTF-8/UTF-8" ];

  boot.initrd.kernelModules = [ "vfat" "nls_cp437" "nls_iso8859_1" "bcache" ];
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

  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
  };

  fonts.packages = with pkgs; [
    noto-fonts
    noto-fonts-cjk-sans
    noto-fonts-cjk-serif
    noto-fonts-color-emoji
  ];

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