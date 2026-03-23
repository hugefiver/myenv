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
  imports =
    [
      # Include the results of the hardware scan.
      #./hardware-configuration.nix

      inputs.disko.nixosModules.disko
      inputs.nixos-facter-modules.nixosModules.facter

      (import ../common/common-pkgs.nix {pkgs = unstable;})
      (import ../common/nix-ld.nix {pkgs = unstable;})

      ./disk.nix
      ./user.nix

      ./web.nix
    ]
    ++ lib.optional (builtins.pathExists /etc/nixos/local/txjp.nix) [/etc/nixos/local/txjp.nix];

  # used for nixos-anywhre installation
  # generate `facter.json` at the root of `flake.nix`
  # hardware.facter.reportPath = "${self}/facter.json";
  hardware.facter.reportPath = ./facter.json;

  # hardware.facter.reportPath =
  #   if builtins.pathExists ./facter.json
  #   then "./facter.json"
  #   else if builtins.pathExists /etc/nixos/local/facter.json
  #   then "/etc/nixos/local/facter.json"
  #   else null;

  # nix.settings.substituters = lib.mkForce [
  #   "https://cache.nixos.org"
  # ];

  nix.settings.experimental-features = ["nix-command" "flakes"];

  # Use the GRUB 2 boot loader.
  boot.loader.grub.enable = true;
  boot.loader.grub.efiSupport = true;
  boot.loader.grub.efiInstallAsRemovable = true;
  # boot.loader.efi.efiSysMountPoint = "/boot/efi";
  # Define on which hard drive you want to install Grub.
  # boot.loader.grub.device = "/dev/sda"; # or "nodev" for efi only

  boot.kernel.sysctl."net.ipv4.tcp_congestion_control" = "bbr";
  boot.kernel.sysctl."net.core.rmem_max" = 16777216;
  boot.kernel.sysctl."net.core.wmem_max" = 16777216;
  boot.kernel.sysctl."net.ipv4.tcp_rmem" = "4096 87380 16777216";
  boot.kernel.sysctl."net.ipv4.tcp_wmem" = "4096 87380 16777216";

  networking.hostName = "nixos-txjp"; # Define your hostname.
  networking.nameservers = [
    "1.0.0.1"
    "8.8.8.8"
    "9.9.9.9"
    # "183.60.83.19"
    # "183.60.82.98"
  ];

  # networking.enableIPv6 = true;
  networking.useDHCP = false;
  # networking.dhcpcd.persistent = true;
  networking.dhcpcd.IPv6rs = false;
  # networking.defaultGateway6 = {
  #   address = "240d:c000:f06f:8e00:8c88:73b4:caa:0";
  #   # gateway = "fd76:3600:201:4f00:0:9e59:b932:d3c5";
  #   interface = "ens3";
  # };
  systemd.network = {
    enable = true;

    networks."10-ens3" = {
      matchConfig.Name = "ens3";

      networkConfig = {
        DHCP = "ipv4";
        IPv6PrivacyExtensions = "kernel";
        IPv6AcceptRA = false;
        # LinkLocalAddressing = "ipv6";
      };

      address = [
        "240d:c000:f06f:9200:8c88:73b4:caa:0"
        # "fd76:3600:201:4f00:0:9f20:c705:c801/128"
      ];

      # gateway = [
      #   "fe80::feee:ffff:feff:ffff"
      # ];

      routes = [
        {
          Destination = "::/0";
          Gateway = "fe80::feee:ffff:feff:ffff";
          GatewayOnLink = true;
          # Metric = 128;
        }
      ];
      linkConfig.RequiredForOnline = "routable";
    };
  };

  environment.variables = {
    EDITOR = "vim";
  };

  # Set your time zone.
  time.timeZone = "Asia/Shanghai";

  environment.systemPackages = with pkgs; [
  ];

  services.openssh = {
    enable = true;
    ports = [2622];
    authorizedKeysInHomedir = true;
    settings = {
      Port = 2622;
      PasswordAuthentication = false;
      #PermitRootLogin = "yes";
      PermitRootLogin = "prohibit-password";
      AllowUsers = ["root" "hugefiver"];

      # LogLevel = "VERBOSE";
    };
  };

  networking.firewall = {
    enable = true;
    allowedTCPPorts = [22 2622 80 443 3478 5349];
    allowedUDPPorts = [443 3478 5349];
    allowedUDPPortRanges = [
      {
        from = 49152;
        to = 65535;
      }
      # { from = 52000; to = 57000; }
    ];
  };

  services.fail2ban = {
    enable = true;

    jails = {
      sshd.settings = {
        enabled = true;
        port = "2622";
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

  services.xray = {
    enable = true;
    settingsFile = "/etc/xray/config.json";
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
