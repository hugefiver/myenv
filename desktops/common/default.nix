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
      ./networking.nix

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
    # 注意：不再覆盖 LC_MESSAGES。Plasma6/KI18n/Qt 都按 LC_MESSAGES 选 UI 翻译，
    # 设成 en_US.UTF-8 会导致 KDE 全英文。让它跟随 i18n.defaultLocale = zh_CN.UTF-8。
  };
  i18n.supportedLocales = [ "zh_CN.UTF-8/UTF-8" "en_US.UTF-8/UTF-8" ];

  console.useXkbConfig = true;
  services.xserver.xkb = {
    layout = "us";
    options = "ctrl:swapcaps";
  };

  environment.sessionVariables = {
    LANG = "zh_CN.UTF-8";
    # LC_MESSAGES 已经从 i18n.extraLocaleSettings 移除，这里也不再强制英文，
    # 否则 Plasma/KDE 应用界面会被强制英文。
    NIXOS_OZONE_WL = "1";
  };

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

  services.flatpak.enable = true;
  systemd.services.flatpak-fcitx5-override = {
    description = "Grant Flatpak apps access to fcitx5 socket";
    wantedBy = [ "multi-user.target" ];
    after = [ "flatpak-system-helper.service" ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
      ExecStart = "${pkgs.flatpak}/bin/flatpak override --filesystem=xdg-run/fcitx5";
    };
  };

  fonts.packages = with pkgs; [
    nerd-fonts.caskaydia-cove
    noto-fonts
    noto-fonts-cjk-sans
    noto-fonts-cjk-serif
    noto-fonts-color-emoji
  ];

  fonts.fontconfig.defaultFonts = {
    monospace = [
      "CaskaydiaCove Nerd Font Mono"
    ];
    emoji = [
      "Noto Color Emoji"
    ];
  };

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