# Edit this configuration file to define what should be installed on
# your system. Help is available in the configuration.nix(5) man page, on
# https://search.nixos.org/options and in the NixOS manual (`nixos-help`).
{
  self,
  inputs,
  lib,
  pkgs,
  unstable,
  system,
  ...
}: {
  imports = [
    # Include the results of the hardware scan.
    #./hardware-configuration.nix

    inputs.disko.nixosModules.disko
    inputs.nixos-facter-modules.nixosModules.facter

    (import ../common/common-pkgs.nix {pkgs = unstable;})
    (import ../common/nix-ld.nix {pkgs = unstable;})

    ./disk.nix
    ./network.nix
    ./user.nix

    ./web.nix
  ];

  # used for nixos-anywhre installation
  # generate `facter.json` at the root of `flake.nix`
  # facter.reportPath = "${self}/facter.json";
  facter.reportPath = ./facter.json;

  # facter.reportPath =
  #   if builtins.pathExists ./facter.json
  #   then "./facter.json"
  #   else if builtins.pathExists /etc/nixos/local/facter.json
  #   then "/etc/nixos/local/facter.json"
  #   else null;

  # nix.settings.substituters = lib.mkForce [
  #   "https://cache.nixos.org"
  # ];

  nix.settings.experimental-features = lib.mkAfter ["nix-command" "flakes"];

  # Use the GRUB 2 boot loader.
  boot.loader.grub.enable = true;
  boot.loader.grub.efiSupport = true;
  boot.loader.grub.efiInstallAsRemovable = true;
  # boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.efi.efiSysMountPoint = "/boot/efi";
  # Define on which hard drive you want to install Grub.
  boot.loader.grub.device = "/dev/vda"; # or "nodev" for efi only
  boot.supportedFilesystems = ["btrfs"];

  boot.kernel.sysctl."net.ipv4.tcp_congestion_control" = "bbr";
  boot.kernel.sysctl."net.core.rmem_max" = 16777216;
  boot.kernel.sysctl."net.core.wmem_max" = 16777216;
  boot.kernel.sysctl."net.ipv4.tcp_rmem" = "4096 87380 16777216";
  boot.kernel.sysctl."net.ipv4.tcp_wmem" = "4096 87380 16777216";

  networking.hostName = "nixos-ccus"; # Define your hostname.
  networking.nameservers = [
    "1.0.0.1"
    "8.8.8.8"
    "9.9.9.9"
    # "183.60.83.19"
    # "183.60.82.98"
  ];

  networking.enableIPv6 = true;
  networking.useDHCP = true;
  # networking.dhcpcd.persistent = true;
  # networking.dhcpcd.wait = "ipv4";
  # networking.dhcpcd.IPv6rs = false;

  environment.variables = {
    EDITOR = "vim";
  };

  # Set your time zone.
  time.timeZone = "Asia/Shanghai";

  environment.systemPackages = lib.mkAfter (with pkgs; [
  ]);

  # services.qemuGuest.enable = true;

  services.openssh = {
    enable = true;
    ports = [2422];
    settings = {
      # Port = 2422;
      PasswordAuthentication = false;
      #PermitRootLogin = "yes";
      PermitRootLogin = "prohibit-password";
      AllowUsers = ["root" "hugefiver"];

      # LogLevel = "VERBOSE";
    };
  };

  networking.firewall = {
    enable = true;
    allowedTCPPorts = [22 2422 80 443];
    allowedUDPPorts = [443];
    allowedUDPPortRanges = [
      # { from = 49152; to = 65535; }
      # { from = 52000; to = 57000; }
    ];
  };

  services.fail2ban = {
    enable = true;

    jails = {
      sshd.settings = {
        enabled = true;
        port = "2422";
      };
    };
  };

  services.logrotate.enable = true;

  services.journald.extraConfig = ''
    Compress=yes
  '';

  virtualisation.docker = {
    enable = true;
    storageDriver = "btrfs";
    liveRestore = true;
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
  system.stateVersion = "25.05"; # Did you read the comment?
}
