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

  # ── SDDM: kwin 自动发现 GPU + 只保留一个输出 ────────────────
  services.displayManager.sddm.settings.Wayland.CompositorCommand = let
    kwin = lib.getExe' pkgs.kdePackages.kwin "kwin_wayland";
    kscreenDoctor = lib.getExe' pkgs.kdePackages.libkscreen "kscreen-doctor";
    udevadm = lib.getExe' pkgs.systemd "udevadm";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_NO_DIRECT_SCANOUT=1

    ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard &
    KWIN_PID=$!

    (
      find_socket() {
        for _s in "$XDG_RUNTIME_DIR"/wayland-*; do
          [ -S "$_s" ] && WAYLAND_DISPLAY="$(basename "$_s")" && export WAYLAND_DISPLAY && return 0
        done
        return 1
      }

      keep_single_output() {
        find_socket || return
        ${kscreenDoctor} -o 2>/dev/null | awk '
          /^Output:/ && /enabled/ { names[n++] = $3 }
          END { for (i = 1; i < n; i++) print names[i] }
        ' | while read -r out; do
          ${kscreenDoctor} "output.$out.disable" 2>/dev/null || true
        done
      }

      sleep 3
      keep_single_output

      ${udevadm} monitor --subsystem-match=drm --kernel 2>/dev/null | while read -r _ts _action _rest; do
        case "$_action" in change|add)
          sleep 2
          keep_single_output
          ;;
        esac
      done
    ) &
    WATCHER_PID=$!

    wait $KWIN_PID
    kill $WATCHER_PID 2>/dev/null
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
