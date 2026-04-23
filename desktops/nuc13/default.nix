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

  programs.clash-verge = {
    enable = true;
    package = unstable.clash-verge-rev;
    tunMode = true;
    serviceMode = true;
  };
  systemd.services.clash-verge.serviceConfig = {
    RuntimeDirectoryMode = lib.mkForce "0755";
    Group = lib.mkForce "users";
  };
  # 同时覆盖 mihomo 默认名 "Meta" 与 GUI 改后的 "Mihomo"
  networking.firewall = {
    trustedInterfaces = [ "Meta" "Mihomo" ];
    extraReversePathFilterRules = ''
      iifname { "Meta", "Mihomo" } accept comment "clash-verge TUN"
    '';
    allowedUDPPorts = [ 4242 ];  # lan-mouse
  };

  # 用户级 handoff 单元免 sudo 管理
  security.polkit.extraConfig = ''
    polkit.addRule(function(action, subject) {
      if (action.id == "org.freedesktop.systemd1.manage-units" &&
          subject.user == "hugefiver") {
        var unit = action.lookup("unit");
        if (unit == "mihomo-boot.service") {
          return polkit.Result.YES;
        }
      }
    });
  '';

  # 登录前透明代理；桌面起来由 mihomo-boot-handoff 同步停止，verge GUI 接管；NAT/路由全交给 mihomo 内部。
  systemd.services.mihomo-boot =
    let
      vergeDir = "/home/hugefiver/.local/share/io.github.clash-verge-rev.clash-verge-rev";
      vergeCfg = "${vergeDir}/clash-verge.yaml";
      startScript = pkgs.writeShellApplication {
        name = "mihomo-boot-start";
        runtimeInputs = [ unstable.mihomo pkgs.iproute2 pkgs.nftables pkgs.gnugrep pkgs.yq-go ];
        text = ''
          VERGE_DIR='${vergeDir}'
          VERGE_CFG='${vergeCfg}'
          RUN_DIR="''${RUNTIME_DIRECTORY:-/run/mihomo-boot}"
          RUN_CFG="$RUN_DIR/config.yaml"

          # 等非-TUN default 路由（最多 60s）
          for i in $(seq 1 60); do
            if ip -4 route show default 2>/dev/null \
                | grep -E '^default ' \
                | grep -vE 'dev (Meta|Mihomo)' \
                | grep -q .; then
              echo "[mihomo-boot] default route ready after ''${i}s"
              break
            fi
            sleep 1
          done

          # 窄路由（仅 fake-ip 段进 TUN）保 SSH 入站回程不被 TUN 劫持；
          # auto-redirect 装 nftables 让 DNS/127.0.0.1 走 mihomo 自己处理。
          yq '
            .tun.enable = true
            | .tun.auto-route = true
            | .tun.auto-redirect = true
            | .tun.inet4-route-address = ["198.18.0.0/16"]
            | .tun.inet6-route-address = ["fc00::/18"]
            | del(.tun.inet4-route-exclude-address)
            | del(.tun.inet6-route-exclude-address)
          ' "$VERGE_CFG" > "$RUN_CFG"

          mihomo -t -d "$VERGE_DIR" -f "$RUN_CFG"
          exec mihomo -d "$VERGE_DIR" -f "$RUN_CFG"
        '';
      };
      # 兜底：mihomo SIGKILL/崩溃时清残留；正常 SIGTERM mihomo 自己会清
      teardown = pkgs.writeShellApplication {
        name = "mihomo-boot-teardown";
        runtimeInputs = [ pkgs.iproute2 ];
        text = ''
          ip link del Mihomo 2>/dev/null || true
          ip link del Meta   2>/dev/null || true
          for p in 9000 9001 9002 9010; do
            ip rule del priority "$p" 2>/dev/null || true
          done
          ip route flush table 2022 2>/dev/null || true
        '';
      };
    in
    {
      description = "mihomo (boot-time, handed off to clash-verge GUI after login)";
      wantedBy = [ "multi-user.target" ];
      unitConfig = {
        ConditionPathExists = vergeCfg;
        StartLimitIntervalSec = 0;
      };
      serviceConfig = {
        Type = "simple";
        ExecStart = "${startScript}/bin/mihomo-boot-start";
        ExecStopPost = "${teardown}/bin/mihomo-boot-teardown";
        Restart = "on-failure";
        RestartSec = "5s";
        TimeoutStartSec = "90s";
        RuntimeDirectory = "mihomo-boot";
        RuntimeDirectoryMode = "0700";
      };
    };
  boot.kernelPackages = pkgs.linuxPackages_zen;
  boot.kernel.features = {
    gcc-x86_64-v3 = true;
  };

  # ── 蓝牙 ─────────────────────────────────────────────────────
  # Plasma6 的 bluedevil 是按 hardware.bluetooth.enable 条件加入的，
  # plasma6.enable 自身不会启用 BlueZ；必须显式打开下面这一项。
  hardware.bluetooth = {
    enable = true;
    powerOnBoot = true;
    settings.General.Experimental = true;  # 启用 BLE 电量等实验特性
  };
  # blueman 提供 obex 文件传输代理 + GTK 配对/管理 GUI（blueman-manager）。
  # Hyprland 会话下作为蓝牙 UI 入口；KDE 会话仍走 bluedevil。
  services.blueman.enable = true;

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
    dirname = "${pkgs.coreutils}/bin/dirname";
    basename = "${pkgs.coreutils}/bin/basename";
    grep = "${pkgs.gnugrep}/bin/grep";
    od = "${pkgs.coreutils}/bin/od";
    mkfifo = "${pkgs.coreutils}/bin/mkfifo";
    rm = "${pkgs.coreutils}/bin/rm";
    udevadm = "${pkgs.systemd}/bin/udevadm";
    pgrep = "${pkgs.procps}/bin/pgrep";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_NO_DIRECT_SCANOUT=1
    _FIFO="/tmp/sddm-drm-monitor.$$"
    _log() { echo "[sddm-compositor] $*" >&2; }

    # ── EDID preferred timing 分辨率检测：原生宽 > 高 = 横屏 ──
    # EDID 第一个 Detailed Timing Descriptor 起始于 byte 54（共 18 字节）：
    #   byte 54-55: pixel clock
    #   byte 56:    H active pixels [7:0]
    #   byte 57:    H blanking [7:0]
    #   byte 58:    H active [11:8] (高 4 位) | H blanking [11:8] (低 4 位)
    #   byte 59:    V active lines [7:0]
    #   byte 60:    V blanking [7:0]
    #   byte 61:    V active [11:8] (高 4 位) | V blanking [11:8] (低 4 位)
    _is_landscape() {
      [ -s "$1" ] || return 1
      local _hlo _hhi _vlo _vhi
      _hlo=$(${od} -A n -t u1 -j 56 -N 1 "$1" 2>/dev/null)
      _hhi=$(${od} -A n -t u1 -j 58 -N 1 "$1" 2>/dev/null)
      _vlo=$(${od} -A n -t u1 -j 59 -N 1 "$1" 2>/dev/null)
      _vhi=$(${od} -A n -t u1 -j 61 -N 1 "$1" 2>/dev/null)
      local _w=$(( ((''${_hhi:-0} >> 4 & 0x0F) << 8) | ''${_hlo:-0} ))
      local _h=$(( ((''${_vhi:-0} >> 4 & 0x0F) << 8) | ''${_vlo:-0} ))
      [ "$_w" -gt "$_h" ]
    }

    # ── 查找最佳显示卡（所有卡同等对待，只看 connected 状态） ──
    # 优先级：1) 已知序列号  2) 横屏  3) 竖屏
    _find_best_card() {
      local _dir _conn _card _st _v _edid
      local _first_landscape="" _first_portrait=""

      for _st in /sys/class/drm/card*-*/status; do
        [ -f "$_st" ] || continue
        read -r _v < "$_st" 2>/dev/null
        [ "$_v" = "connected" ] || continue
        _dir=$(${dirname} "$_st")
        _conn=$(${basename} "$_dir")
        _card="''${_conn%%-*}"
        _edid="$_dir/edid"

        # 1) 已知序列号直接返回
        if [ -s "$_edid" ] && ${grep} -qa "60PZCH3" "$_edid" 2>/dev/null; then
          _log "selected $_card ($_conn): matched serial 60PZCH3"
          echo "$_card"
          return 0
        fi

        # 2) 横屏  3) 竖屏
        if [ -z "$_first_landscape" ] && _is_landscape "$_edid"; then
          _first_landscape="$_card"
        elif [ -z "$_first_portrait" ]; then
          _first_portrait="$_card"
        fi
      done

      if [ -n "$_first_landscape" ]; then
        _log "selected $_first_landscape: first landscape display"
        echo "$_first_landscape"; return 0
      fi
      if [ -n "$_first_portrait" ]; then
        _log "selected $_first_portrait: first portrait display (no landscape found)"
        echo "$_first_portrait"; return 0
      fi
      _log "no connected display found"
      return 1
    }

    # ── 1. 即时检测 ──
    _card=$(_find_best_card)

    # ── 2. 无显示设备 → 事件驱动等待（非轮询） ──
    if [ -z "$_card" ]; then
      _log "waiting for DRM device (event-driven)..."
      while IFS= read -r _; do
        sleep 1
        while IFS= read -r -t 0.5 _; do :; done
        _card=$(_find_best_card)
        [ -n "$_card" ] && break
      done < <(${udevadm} monitor -s drm -u)
    fi

    # 兜底：事件流异常退出仍未找到设备 → 不设 KWIN_DRM_DEVICES，让 kwin 自行探测
    if [ -z "$_card" ]; then
      _log "WARNING: no display found after event wait, letting kwin auto-detect"
    # ── 3. KWIN_DRM_DEVICES：iGPU 当 render，选中的卡当 scanout ──
    # 关键：永远用 udev symlink /dev/dri/intel-igpu 当 render node，不要赌 card 编号。
    # evdi 启动顺序不固定，有概率 card0 = evdi。evdi 没有 renderD*（虚拟扫描出口、
    # 没 GPU 单元），单独喂给 kwin 起不来 → SDDM "缩小动画" 后停在最后一帧。
    elif [ -e /dev/dri/intel-igpu ]; then
      export KWIN_DRM_DEVICES="/dev/dri/intel-igpu:/dev/dri/$_card"
    else
      export KWIN_DRM_DEVICES="/dev/dri/$_card"
    fi
    _log "KWIN_DRM_DEVICES=''${KWIN_DRM_DEVICES:-<auto>}"

    # ── 4. kwin（后台子进程，主进程保留控制权） ──
    ${kwin} --drm --no-lockscreen --no-global-shortcuts --inputmethod qtvirtualkeyboard &
    _KWIN_PID=$!

    # ── 5. DRM 设备变更监听 ──
    ${rm} -f "$_FIFO"
    ${mkfifo} "$_FIFO"
    ${udevadm} monitor -s drm -u > "$_FIFO" 2>/dev/null &
    _UDEV_PID=$!

    _cleanup() { kill "$_KWIN_PID" "$_UDEV_PID" 2>/dev/null; ${rm} -f "$_FIFO"; }
    trap '_cleanup; wait "$_KWIN_PID" 2>/dev/null; exit' TERM INT HUP

    _CUR="$_card"
    while true; do
      if ! IFS= read -r -t 5 _; then
        kill -0 "$_KWIN_PID" 2>/dev/null || break
        # udevadm 进程也挂了 → 没法继续监听
        kill -0 "$_UDEV_PID" 2>/dev/null || { _log "udevadm exited, stopping monitor"; break; }
        continue
      fi
      # DRM 事件 → 防抖 2s
      sleep 2
      while IFS= read -r -t 0.5 _; do :; done
      kill -0 "$_KWIN_PID" 2>/dev/null || break
      _new=$(_find_best_card)
      if [ "''${_new:-}" != "$_CUR" ]; then
        _log "display changed: $_CUR -> ''${_new:-<none>}, restarting compositor"
        ${pgrep} -x sddm-greeter-qt6 >/dev/null 2>&1 && kill "$_KWIN_PID"
        break
      fi
    done < "$_FIFO"

    _cleanup
    wait "$_KWIN_PID" 2>/dev/null
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
