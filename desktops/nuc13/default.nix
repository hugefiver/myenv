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
  ];
  boot.initrd.kernelModules = [
    "dm-cache"
    "dm-cache-smq"
  ];

  services.displayManager.sddm.settings.Wayland.CompositorCommand = let
    kwin = lib.getExe' pkgs.kdePackages.kwin "kwin_wayland";
    kscreenDoctor = lib.getExe' pkgs.kdePackages.libkscreen "kscreen-doctor";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_NO_DIRECT_SCANOUT=1

    # ── 尝试限制 kwin 只用 iGPU 直连输出 ──
    if [ -e /dev/dri/intel-igpu ]; then
      _igpu=$(readlink -f /dev/dri/intel-igpu)
      # 检查 iGPU 是否有连接的输出（status=connected）
      _card_name="''${_igpu##*/}"
      _has_conn=0
      for _s in /sys/class/drm/"$_card_name"-*/status; do
        [ -f "$_s" ] && [ "$(cat "$_s")" = "connected" ] && _has_conn=1 && break
      done
      if [ "$_has_conn" = "1" ]; then
        export KWIN_DRM_DEVICES="$_igpu"
        exec ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard
      fi
    fi

    # ── iGPU 无输出，使用所有 DRM 设备，登录后选择最佳单输出 ──
    ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard &
    _PID=$!

    (
      # 等待至少 1 个输出出现
      for _i in $(seq 1 15); do
        _cnt=$(${kscreenDoctor} -o 2>/dev/null | grep -c "^Output:" || true)
        [ "$_cnt" -ge 1 ] && break
        sleep 1
      done
      sleep 2

      # 解析所有输出，按优先级选择：横屏 > 第一竖屏
      _keep="" _first=""
      _id="" _w=0 _h=0
      _eval_output() {
        [ -z "$_id" ] && return
        [ -z "$_first" ] && _first="$_id"
        # 横屏（w >= h）优先
        if [ -z "$_keep" ] && [ "$_w" -ge "$_h" ] && [ "$_w" -gt 0 ]; then
          _keep="$_id"
        fi
      }
      while IFS= read -r _line; do
        case "$_line" in
          Output:*)
            _eval_output
            set -- $_line; _id="$2"; _w=0; _h=0
            ;;
          *Geometry:*)
            _res="''${_line##* }"
            _w="''${_res%%x*}"
            _h="''${_res##*x}"
            ;;
        esac
      done < <(${kscreenDoctor} -o 2>/dev/null)
      _eval_output

      # 没找到横屏就回退到第一个输出
      [ -z "$_keep" ] && _keep="$_first"

      # 禁用所有非 _keep 的输出
      while IFS= read -r _line; do
        case "$_line" in
          Output:*)
            set -- $_line
            [ "$2" != "$_keep" ] && ${kscreenDoctor} output."$2".disable 2>/dev/null || true
            ;;
        esac
      done < <(${kscreenDoctor} -o 2>/dev/null)
    ) &

    wait $_PID
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
    WLR_NO_HARDWARE_CURSORS = "1";   # DisplayLink USB 链路下避免光标异常
  };

  services.openssh = {
    enable = true; 
    authorizedKeysInHomedir = true;
    settings = {
      AllowUsers = ["hugefiver"];
    };
  };
}
