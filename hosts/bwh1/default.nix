{
  self,
  inputs,
  lib,
  pkgs,
  system,
  nixpkgs,
  nixpkgs-unstable,
  ...
}: let
  unstable = import nixpkgs-unstable {inherit system;};
in {
  imports =
    [
      inputs.disko.nixosModules.disko
      inputs.nixos-facter-modules.nixosModules.facter

      (import ../common/common-pkgs.nix {pkgs = unstable;})
      ./disk.nix

      ./user.nix
    ]
    ++ lib.optional (builtins.pathExists /etc/nixos/local/bwh1.nix) [/etc/nixos/local/bwh1.nix];

  facter.reportPath =
    if builtins.pathExists ./facter.json
    then "./facter.json"
    else if builtins.pathExists /etc/nixos/local/facter.json
    then "/etc/nixos/local/facter.json"
    else null;

  # nix.settings.substituters = lib.mkForce [
  #   "https://cache.nixos.org"
  # ];

  nix.settings.experimental-features = ["nix-command" "flakes"];

  # Use the GRUB 2 boot loader.
  boot.loader.grub.enable = true;
  # boot.loader.grub.efiSupport = true;
  # boot.loader.grub.efiInstallAsRemovable = true;
  # boot.loader.efi.efiSysMountPoint = "/boot/efi";
  # Define on which hard drive you want to install Grub.
  # boot.loader.grub.device = "/dev/sda"; # or "nodev" for efi only

  networking.hostName = "bwh1"; # Define your hostname.
  networking.nameservers = [
    "1.0.0.1"
    "8.8.8.8"
    "9.9.9.9"
  ];

  environment.variables = {
    EDITOR = "vim";
  };

  # Set your time zone.
  time.timeZone = "Asia/Shanghai";
  # i18n.defaultLocale = "en_US.UTF-8";

  environment.systemPackages =
    (with pkgs; [
      ])
    ++ (
      with unstable; [
        rclone
        jdk21
      ]
    );

  services.openssh = {
    enable = true;
    settings = {
      PasswordAuthentication = false;
      PermitRootLogin = "prohibit-password";
      AllowUsers = ["root" "hugefiver"];

      # LogLevel = "VERBOSE";
    };
  };

  networking.firewall = {
    enable = true;
    allowedTCPPorts = [22];
    allowedUDPPorts = [];
  };

  services.fail2ban = {
    enable = true;
  };

  virtualisation.docker = {
    enable = true;
    storageDriver = "btrfs";
  };

  users.extraGroups.docker.members = ["root" "hugefiver"];

  security.sudo = {
    enable = true;
    wheelNeedsPassword = false;
  };

  # Copy the NixOS configuration file and link it from the resulting system
  # (/run/current-system/configuration.nix). This is useful in case you
  # accidentally delete configuration.nix.
  # system.copySystemConfiguration = true;

  # This option defines the first version of NixOS you have installed on this particular machine,
  # and is used to maintain compatibility with application data (e.g. databases) created on older NixOS versions.
  #
  # Most users should NEVER change this value after the initial install, for any reason,
  # even if you've upgraded your system to a new NixOS release.
  #
  # This value does NOT affect the Nixpkgs version your packages and OS are pulled from,
  # so changing it will NOT upgrade your system - see https://nixos.org/manual/nixos/stable/#sec-upgrading for how
  # to actually do that.
  #
  # This value being lower than the current NixOS release does NOT mean your system is
  # out of date, out of support, or vulnerable.
  #
  # Do NOT change this value unless you have manually inspected all the changes it would make to your configuration,
  # and migrated your data accordingly.
  #
  # For more information, see `man configuration.nix` or https://nixos.org/manual/nixos/stable/options#opt-system.stateVersion .
  system.stateVersion = "24.11"; # Did you read the comment?
}
