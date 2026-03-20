args@{ self, inputs, lib, pkgs, ... }:
let
  flakeShare = "/mnt/flake";
  envPubKeyPath = builtins.getEnv "DESKTOP_NUC13_TEST_PUBKEY_PATH";
  desktopNuc13TestPubKey = args.desktopNuc13TestPubKey or (if envPubKeyPath != "" then builtins.readFile envPubKeyPath else builtins.readFile ./desktop-nuc13-test-key.pub);
in {
  imports = [
    inputs.microvm.nixosModules.microvm
  ];

  networking.hostName = "desktop-nuc13-installer-vm";
  system.stateVersion = "25.11";

  services.openssh.enable = true;
  users.users.root.openssh.authorizedKeys.keys = [
    desktopNuc13TestPubKey
  ];
  security.sudo.wheelNeedsPassword = false;

  services.getty.autologinUser = "root";

  networking.useDHCP = lib.mkDefault true;
  systemd.network.wait-online.enable = false;

  nix.settings.experimental-features = [ "nix-command" "flakes" ];
  nix.channel.enable = false;

  boot.loader.systemd-boot.enable = false;
  boot.loader.grub.enable = false;

  fileSystems."/" = lib.mkForce {
    device = "rootfs";
    fsType = "tmpfs";
    options = [ "size=80%" "mode=0755" ];
    neededForBoot = true;
  };

  environment.systemPackages = with pkgs; [
    coreutils
    gnugrep
    lvm2
    util-linux
    btrfs-progs
  ];

  boot.kernelParams = [
    "console=ttyS0,115200"
    "rd.systemd.show_status=1"
  ];
  boot.kernelModules = [
    "dm-cache"
    "dm-cache-smq"
    "dm-cache-mq"
    "dm-cache-cleaner"
  ];

  microvm = {
    hypervisor = "qemu";
    mem = 8192;
    vcpu = 4;
    cpu = "max";
    socket = "desktop-nuc13-installer.sock";
    shares = [
      {
        tag = "ro-store";
        source = "/nix/store";
        mountPoint = "/nix/.ro-store";
      }
      {
        tag = "flake-src";
        source = toString self;
        mountPoint = flakeShare;
      }
      {
        tag = "results";
        source = "results";
        mountPoint = "/mnt/results";
      }
    ];
    interfaces = [
      {
        type = "user";
        id = "net0";
        mac = "02:00:00:00:10:01";
      }
    ];
    forwardPorts = [
      {
        from = "host";
        host.address = "127.0.0.1";
        host.port = 2222;
        guest.port = 22;
      }
    ];
    qemu.machine = "q35";
    qemu.machineOpts = {
      accel = "tcg";
      acpi = "on";
      mem-merge = "on";
    };
    qemu.extraArgs = [
      "-drive" "if=pflash,format=raw,readonly=on,file=${pkgs.OVMF.fd}/FV/OVMF_CODE.fd"
      "-drive" "if=pflash,format=raw,file=desktop-nuc13-installer-vars.fd"
      "-drive" "if=none,id=nvme0,file=desktop-nuc13-cache.img,format=raw"
      "-device" "nvme,serial=deadbeef,drive=nvme0"
      "-drive" "if=none,id=hdd0,file=desktop-nuc13-main.img,format=raw"
      "-device" "virtio-scsi-pci,id=scsi0"
      "-device" "scsi-hd,drive=hdd0,bus=scsi0.0"
    ];
    preStart = ''
      mkdir -p results
      if [ ! -f desktop-nuc13-cache.img ]; then
        truncate -s 4G desktop-nuc13-cache.img
      fi
      if [ ! -f desktop-nuc13-main.img ]; then
        truncate -s 12G desktop-nuc13-main.img
      fi
      if [ ! -f desktop-nuc13-installer-vars.fd ]; then
        cp ${pkgs.OVMF.fd}/FV/OVMF_VARS.fd desktop-nuc13-installer-vars.fd
      fi
      chmod u+w desktop-nuc13-installer-vars.fd
    '';
  };

}
