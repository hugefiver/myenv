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

  # ── SDDM: 即时启动 + DisplayLink 热插拔 + 单横屏登录 ────────────
  # kwin 立即启动（不等待 evdi），自动发现当前可用 GPU。
  # DisplayLink 设备后续出现时 kwin 通过 DRM uevent 自动检测。
  # 后台守护监控输出变化：只保留一个横屏，禁用竖屏。
  services.displayManager.sddm.settings.Wayland.CompositorCommand = let
    kwin = lib.getExe' pkgs.kdePackages.kwin "kwin_wayland";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_NO_DIRECT_SCANOUT=1

    # 后台守护：监控输出变化，确保 SDDM 只在一个横屏上显示
    (
      # 等待 kwin Wayland socket 就绪
      for _i in $(seq 1 50); do
        for _s in "$XDG_RUNTIME_DIR"/wayland-*; do
          [ -S "$_s" ] && export WAYLAND_DISPLAY=$(basename "$_s") && break 2
        done
        sleep 0.1
      done

      enforce_single_landscape() {
        # 获取所有已连接输出及其分辨率
        local outputs
        outputs=$(kscreen-doctor -o 2>/dev/null) || return

        local landscape="" portrait=""
        local cur_name="" cur_w=0 cur_h=0

        while IFS= read -r line; do
          case "$line" in
            Output:*)
              # 处理上一个输出
              if [ -n "$cur_name" ] && [ "$cur_w" -gt 0 ]; then
                if [ "$cur_h" -gt "$cur_w" ]; then
                  portrait="$portrait $cur_name"
                else
                  landscape="$landscape $cur_name"
                fi
              fi
              cur_name=$(echo "$line" | awk '{print $3}')
              cur_w=0; cur_h=0
              ;;
            *Geometry:*)
              local wh=$(echo "$line" | awk '{print $NF}')
              cur_w=$(echo "$wh" | cut -dx -f1)
              cur_h=$(echo "$wh" | cut -dx -f2)
              ;;
          esac
        done <<< "$outputs"
        # 处理最后一个
        if [ -n "$cur_name" ] && [ "$cur_w" -gt 0 ]; then
          if [ "$cur_h" -gt "$cur_w" ]; then
            portrait="$portrait $cur_name"
          else
            landscape="$landscape $cur_name"
          fi
        fi

        # 禁用所有竖屏
        for _out in $portrait; do
          kscreen-doctor "output.$_out.disable" 2>/dev/null || true
        done

        # 如果有多个横屏，只保留第一个
        local first=true
        for _out in $landscape; do
          if $first; then
            first=false
          else
            kscreen-doctor "output.$_out.disable" 2>/dev/null || true
          fi
        done
      }

      # 初始执行 + 每 2 秒轮询（捕获 DisplayLink 热插拔后的输出变化）
      sleep 0.5
      prev_hash=""
      while true; do
        cur_hash=$(kscreen-doctor -o 2>/dev/null | grep -c "Output:" || echo 0)
        if [ "$cur_hash" != "$prev_hash" ]; then
          sleep 0.3  # 等输出稳定
          enforce_single_landscape
          prev_hash="$cur_hash"
        fi
        sleep 2
      done
    ) >/dev/null 2>&1 &

    exec ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard
  '');

  # dlm.service 不再阻塞 SDDM 启动，改为 Wants（并行启动）
  systemd.services.display-manager.after = lib.mkForce [];
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

  # ── 多 GPU 会话变量（Hyprland / aquamarine）─────────────────────
  # env-hyprland 通过 UWSM source 也会设置这些，这里作为 fallback
  # 确保即使 UWSM 未正确读取 env 文件也能让 Hyprland 找到双 GPU。
  environment.sessionVariables = {
    AQ_DRM_DEVICES = "/dev/dri/intel-igpu:/dev/dri/displaylink-card";
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
