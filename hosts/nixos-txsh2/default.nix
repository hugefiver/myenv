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
    ]
    ++ lib.optional (builtins.pathExists /etc/nixos/local/txsh.nix) [/etc/nixos/local/txsh.nix];

  # used for nixos-anywhre installation
  # generate `facter.json` near this file
  hardware.facter.reportPath = ./facter.json;

  # hardware.facter.reportPath =
  #   if builtins.pathExists ./facter.json
  #   then "./facter.json"
  #   else if builtins.pathExists /etc/nixos/local/facter.json
  #   then "/etc/nixos/local/facter.json"
  #   else null;

  nix.settings.substituters = lib.mkForce [
    "https://mirrors.cernet.edu.cn/nix-channels/store"
    # "https://mirror.sjtu.edu.cn/nix-channels/store"
    # "https://mirrors.sjtug.sjtu.edu.cn/nix-channels/store"
    # "https://mirrors.tuna.tsinghua.edu.cn/nix-channels/store"
    # "https://mirrors.ustc.edu.cn/nix-channels/store"
    "https://cache.nixos.org"
  ];

  nix.settings.experimental-features = ["nix-command" "flakes"];

  # Use the GRUB 2 boot loader.
  boot.loader.grub.enable = true;
  # boot.loader.grub.efiSupport = true;
  # boot.loader.grub.efiInstallAsRemovable = true;
  # boot.loader.efi.efiSysMountPoint = "/boot/efi";
  # Define on which hard drive you want to install Grub.
  # boot.loader.grub.device = "/dev/vda"; # or "nodev" for efi only
  # fileSystems."/".device = lib.mkDefault "/dev/vda";

  boot.kernel.sysctl."net.ipv4.tcp_congestion_control" = "bbr";
  boot.kernel.sysctl."net.core.rmem_max" = 16777216;
  boot.kernel.sysctl."net.core.wmem_max" = 16777216;
  boot.kernel.sysctl."net.ipv4.tcp_rmem" = "4096 87380 16777216";
  boot.kernel.sysctl."net.ipv4.tcp_wmem" = "4096 87380 16777216";

  networking.hostName = "nixos-txsh2";

  # Pick only one of the below networking options.
  # networking.wireless.enable = true;  # Enables wireless support via wpa_supplicant.
  # networking.networkmanager.enable = true;  # Easiest to use and most distros use this by default.
  # networking.nameservers = [
  #   "119.29.29.29"
  #   "223.5.5.5"
  #   "8.8.8.8"
  #   # "183.60.83.19"
  #   # "183.60.82.98"
  # ];

  # networking.enableIPv6 = true;
  networking.useDHCP = false;
  # networking.dhcpcd.persistent = true;
  networking.dhcpcd.IPv6rs = false;

  systemd.network = {
    enable = true;

    networks."10-eth0" = {
      matchConfig.Name = "eth0";

      networkConfig = {
        DHCP = "ipv4";
        IPv6PrivacyExtensions = "kernel";
        IPv6AcceptRA = false;
        # LinkLocalAddressing = "ipv6";
      };

      address = [
        "2402:4e00:c032:6800:8c88:73b4:caa:1"
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

  # services.resolved.enable = false;

  # services.dnsmasq = {
  #   enable = true;
  #   settings = {
  #     server = [
  #       "/tencentyun.com/183.60.83.19"
  #       "/tencentyun.com/183.60.82.98"
  #       "119.29.29.29"
  #       "223.5.5.5"
  #       "8.8.8.8"
  #     ];
  #   };
  # };

  # systemd.services.dnsmasq = {
  #   requires = ["network.target"];
  #   wants = ["systemd-networkd-wait-online.service" "network-online.target"];
  #   after = ["systemd-networkd-wait-online.service" "network-online.target"];
  # };

  environment.variables = {
    EDITOR = "vim";
  };

  # Set your time zone.
  time.timeZone = "Asia/Shanghai";

  environment.systemPackages = with pkgs;
    [
      docker-compose
    ]
    ++ (
      with unstable; [
        restic
      ]
    );

  services.openssh = {
    enable = true;
    ports = [2422];
    authorizedKeysInHomedir = true;
    settings = {
      PasswordAuthentication = false;
      #PermitRootLogin = "yes";
      PermitRootLogin = "prohibit-password";
      AllowUsers = ["root" "hugefiver"];

      # LogLevel = "VERBOSE";
    };
  };

  networking.firewall = {
    enable = true;
    allowedTCPPorts = [22 2422 3478];
    allowedUDPPorts = [3478];
    allowedUDPPortRanges = [
      {
        from = 49152;
        to = 65535;
      }
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
  };
  systemd.services.docker.unitConfig.Restart = "on-failure";

  users.extraGroups.docker.members = ["root" "hugefiver"];

  security.sudo = {
    enable = true;
    wheelNeedsPassword = false;
  };

  services.xray = {
    enable = true;
    settingsFile = "/etc/xray/config.json";
  };

  # Configure network proxy if necessary
  # networking.proxy.default = "http://user:password@proxy:port/";
  # networking.proxy.noProxy = "127.0.0.1,localhost,internal.domain";

  system.stateVersion = "26.05";
}
