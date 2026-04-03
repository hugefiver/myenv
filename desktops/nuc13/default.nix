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

  # ── Clash Verge Rev (TUN 代理) ────────────────────────────────
  programs.clash-verge = {
    enable = true;
    package = unstable.clash-verge-rev;
    tunMode = true;       # setcap cap_net_admin
    serviceMode = true;   # systemd 后台服务
  };
  systemd.services.clash-verge.serviceConfig.RuntimeDirectoryMode = "0755";
  # TUN 接口需要 nftables 放通，否则流量被 rpfilter 丢弃
  # https://github.com/NixOS/nixpkgs/issues/477636
  networking.firewall = {
    trustedInterfaces = [ "Mihomo" ];
    extraReversePathFilterRules = ''
      iifname { "Mihomo" } accept comment "clash-verge TUN"
    '';
  };
  
  boot.kernelPackages = pkgs.linuxPackages_zen;
  boot.kernel.features = {
    gcc-x86_64-v3 = true;
  };

  networking.hostName = "desktop-nuc13";
  # NetworkManager 全权管理网络，scripted backend 不需要管任何接口。
  # networking.useDHCP = false 仅关闭全局默认；但 nixos-facter 会为
  # facter.json 中检测到的每个接口生成 useDHCP = mkDefault true，
  # 导致 scripted backend 仍创建 BindsTo=sys-subsystem-net-devices-wlo1.device
  # 的服务单元——WiFi 固件加载慢时卡 90s。
  # mkForce 清空 interfaces 确保 scripted backend 不生成任何接口服务。
  networking.useDHCP = false;
  networking.interfaces = lib.mkForce {};
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
    "iwlmvm"
  ];
  boot.initrd.kernelModules = [
    "dm-cache"
    "dm-cache-smq"
  ];

  services.displayManager.sddm.settings.Wayland.CompositorCommand = let
    kwin = lib.getExe' pkgs.kdePackages.kwin "kwin_wayland";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_NO_DIRECT_SCANOUT=1

    # ── 通过 EDID 序列号找到横屏(60PZCH3)所在的 DRM card ──
    _find_landscape_card() {
      for _edid in /sys/class/drm/card*-*/edid; do
        [ -f "$_edid" ] || continue
        if ${pkgs.coreutils}/bin/strings "$_edid" 2>/dev/null | ${pkgs.gnugrep}/bin/grep -q "60PZCH3"; then
          _dir=$(${pkgs.coreutils}/bin/dirname "$_edid")
          _conn=$(${pkgs.coreutils}/bin/basename "$_dir")
          echo "''${_conn%%-*}"
          return 0
        fi
      done
      return 1
    }

    # 等待 DisplayLink/evdi 初始化（最多 30s）
    _card=""
    for _i in $(seq 1 30); do
      _card=$(_find_landscape_card) && break
      sleep 1
    done

    if [ -n "$_card" ]; then
      _dev="/dev/dri/$_card"
      if [ -e /dev/dri/intel-igpu ]; then
        _igpu=$(${pkgs.coreutils}/bin/readlink -f /dev/dri/intel-igpu)
        export KWIN_DRM_DEVICES="$_igpu:$_dev"
      else
        export KWIN_DRM_DEVICES="$_dev"
      fi
    fi

    exec ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard
  '');

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

  # ── 多 GPU 会话变量（Hyprland / aquamarine）─────────────────────
  # AQ_DRM_DEVICES 由 env-hyprland 动态发现所有 card 设备设置，
  # 这里只设不依赖设备枚举的静态标志。
  environment.sessionVariables = {
    AQ_MGPU_NO_EXPLICIT = "1";       # evdi 不支持 explicit sync
  };

  services.openssh = {
    enable = true; 
    authorizedKeysInHomedir = true;
    settings = {
      AllowUsers = ["hugefiver"];
    };
  };
}
