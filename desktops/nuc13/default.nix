{
  self,
  config,
  inputs,
  lib,
  pkgs,
  unstable,
  system,
  ...
} : {
  imports = [
    ../common
    ../common/personal.nix
    ../common/hyprland.nix
    ../common/kde.nix

    ./disk.nix
  ];
  
  boot.kernelPackages = pkgs.linuxPackages_zen;
  boot.kernel.features = {
    gcc-x86_64-v3 = true;
  };

  networking.hostName = "desktop-nuc13";
  hardware.facter.reportPath = ./facter.json;
  hardware.enableRedistributableFirmware = true;

  systemd.network.wait-online.enable = false;

  nixpkgs.config.allowUnfree = true;
  services.xserver.videoDrivers = [ "displaylink" "modesetting" ];
  services.lvm.boot.thin.enable = true;

  boot.extraModulePackages = [ config.boot.kernelPackages.evdi ];
  boot.extraModprobeConfig = ''
    options evdi initial_device_count=0
  '';
  boot.kernelModules = [
    "evdi"
  ];
  boot.initrd.kernelModules = [
    "dm-cache"
    "dm-cache-smq"
  ];

  # ── SDDM: 仅在 Intel iGPU 渲染登录界面 ────────────────────────────
  # 覆写 SDDM 的 Wayland compositor 命令，通过 KWIN_DRM_DEVICES 环境
  # 变量限制 kwin_wayland 仅使用 Intel iGPU；这样 DisplayLink 输出
  # 不会参与 SDDM 渲染，解决登录界面卡顿和多显示器重复显示的问题。
  # 用户会话的 KWIN_DRM_DEVICES 在 kde.nix 的 sessionVariables 中单独设置，
  # 包含两块 GPU，因此登录后 KDE/kwin 能正常驱动所有显示器。
  services.displayManager.sddm.settings.Wayland.CompositorCommand = let
    kwin = lib.getExe' pkgs.kdePackages.kwin "kwin_wayland";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_DEVICES=/dev/dri/intel-igpu
    export KWIN_DRM_NO_DIRECT_SCANOUT=1
    exec ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard
  '');

  # 确保 DisplayLink 管理器在 SDDM 之前就绪，这样用户登录时
  # evdi DRM 设备已经存在，Hyprland/KDE 可以正确识别所有显示器。
  systemd.services.display-manager.after = [ "dlm.service" ];
  systemd.services.display-manager.wants = [ "dlm.service" ];

  nixpkgs.overlays = [
    (final: prev: {
      displaylink = prev.displaylink.overrideAttrs (oldAttrs: {
        src = final.fetchurl {
          url = "https://www.synaptics.com/sites/default/files/exe_files/2025-09/DisplayLink%20USB%20Graphics%20Software%20for%20Ubuntu6.2-EXE.zip";
          name = "displaylink-620.zip";
          hash = "sha256-JQO7eEz4pdoPkhcn9tIuy5R4KyfsCniuw6eXw/rLaYE=";
        };
      });
    })
  ];

  services.udev.extraRules = ''
    ACTION=="add|change", SUBSYSTEM=="drm", KERNEL=="card*", KERNELS=="0000:00:02.0", SUBSYSTEMS=="pci", SYMLINK+="dri/intel-igpu"
    ACTION=="add|change", SUBSYSTEM=="drm", KERNEL=="card*", DRIVERS=="evdi", SYMLINK+="dri/displaylink-card"
  '';

  services.openssh = {
    enable = true; 
    authorizedKeysInHomedir = true;
    settings = {
      AllowUsers = ["hugefiver"];
    };
  };
}
