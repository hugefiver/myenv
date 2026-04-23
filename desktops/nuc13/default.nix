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
  systemd.services.clash-verge.serviceConfig = {
    RuntimeDirectoryMode = lib.mkForce "0755";
    Group = lib.mkForce "users";
  };
  # TUN 接口需放通 rpfilter，否则流量被丢。
  # 同时覆盖 mihomo 默认名 "Meta" 与用户在 GUI 改后的 "Mihomo"。
  # nixpkgs programs.clash-verge 模块未暴露接口名选项，只能两个都列。
  networking.firewall = {
    trustedInterfaces = [ "Meta" "Mihomo" ];
    extraReversePathFilterRules = ''
      iifname { "Meta", "Mihomo" } accept comment "clash-verge TUN"
    '';
    allowedUDPPorts = [ 4242 ];  # lan-mouse

    # ── DNS 重定向到 mihomo ────────────────────────────────
    # mihomo-boot 把 TUN 路由收窄到 fake-ip /16 后 default 不再进 TUN，
    # 系统 DNS（8.8.8.8 之类）不会被 sing-tun 自动 hijack，应用拿到真实 IP →
    # 不命中 198.18/16 → 不进 TUN → 直连失败。
    #
    # 这里把所有出站 :53 NAT 到 mihomo `127.0.0.1:8853` 补上 fake-ip 链路。
    # fwmark 0x6d6968 是 sing-tun 给 mihomo 自身上行 DNS 打的 mark，必须
    # RETURN 跳过否则会自循环。
    extraCommands = ''
      iptables -t nat -F mihomo-dns 2>/dev/null || iptables -t nat -N mihomo-dns
      iptables -t nat -A mihomo-dns -m mark --mark 0x6d6968 -j RETURN
      iptables -t nat -A mihomo-dns -d 127.0.0.0/8 -j RETURN
      iptables -t nat -A mihomo-dns -p udp --dport 53 -j REDIRECT --to-ports 8853
      iptables -t nat -A mihomo-dns -p tcp --dport 53 -j REDIRECT --to-ports 8853
      iptables -t nat -C OUTPUT -j mihomo-dns 2>/dev/null || iptables -t nat -A OUTPUT -j mihomo-dns
    '';
    extraStopCommands = ''
      iptables -t nat -D OUTPUT -j mihomo-dns 2>/dev/null || true
      iptables -t nat -F mihomo-dns 2>/dev/null || true
      iptables -t nat -X mihomo-dns 2>/dev/null || true
    '';
  };

  # ── Boot-time mihomo daemon (handed off to clash-verge after login) ──
  # 用户登录前就把 TUN 代理拉起来。读 verge 上次落盘的运行时 yaml
  # （subscriptions + Merge.yaml + Script.js 合并结果）。
  # 登录到桌面后 autostart.sh 会 `sudo systemctl stop mihomo-boot`，
  # 让出 TUN / 7890 / 9090，clash-verge GUI 接管自己的 mihomo 实例。
  #
  # wrapper 用 yq 把 `tun.inet4-route-address` 收窄到 fake-ip /16，
  # default 不再进 TUN → SSH 入站回包走 main → wlo1 不被 hijack。
  # DNS 链路靠上面 firewall.extraCommands 的 :53 → mihomo NAT 兜底。
  #
  # ⚠ 前提：verge profile 必须开 fake-ip
  #   `dns.enhanced-mode: fake-ip` + `dns.fake-ip-range: 198.18.0.1/16`
  #   `dns.listen: 127.0.0.1:8853`（与上面 NAT 端口对齐）
  systemd.services.mihomo-boot =
    let
      vergeDir = "/home/hugefiver/.local/share/io.github.clash-verge-rev.clash-verge-rev";
      vergeCfg = "${vergeDir}/clash-verge.yaml";
      startScript = pkgs.writeShellApplication {
        name = "mihomo-boot-start";
        runtimeInputs = [ unstable.mihomo pkgs.iproute2 pkgs.gnugrep pkgs.yq-go ];
        text = ''
          VERGE_DIR='${vergeDir}'
          VERGE_CFG='${vergeCfg}'
          RUN_DIR="''${RUNTIME_DIRECTORY:-/run/mihomo-boot}"
          RUN_CFG="$RUN_DIR/config.yaml"

          # 等到出现非-TUN 的 default 路由（NM 起好 wlo1 / eth0），最多 60s
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

          # 收窄 TUN 路由到 fake-ip 段，default 留给主路由表（保住 SSH 等入站）
          yq '
            .tun.inet4-route-address = ["198.18.0.0/16"]
            | .tun.inet6-route-address = ["fc00::/18"]
            | del(.tun.inet4-route-exclude-address)
            | del(.tun.inet6-route-exclude-address)
          ' "$VERGE_CFG" > "$RUN_CFG"

          # 校验配置（坏 yaml 直接退出，systemd 5s 后重试）
          mihomo -t -d "$VERGE_DIR" -f "$RUN_CFG"

          echo "[mihomo-boot] starting mihomo (TUN narrowed to fake-ip range)"
          exec mihomo -d "$VERGE_DIR" -f "$RUN_CFG"
        '';
      };
    in
    {
      description = "Mihomo proxy (boot-time, fake-ip TUN, handed off to clash-verge after login)";
      wantedBy = [ "multi-user.target" ];
      unitConfig = {
        ConditionPathExists = vergeCfg;
        StartLimitIntervalSec = 0;  # 无限重试，配合下面 RestartSec=5s
      };
      serviceConfig = {
        Type = "simple";
        ExecStart = "${startScript}/bin/mihomo-boot-start";
        Restart = "on-failure";
        RestartSec = "5s";
        # 60s wait + 启动余量
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

    # ── 查找首选显示卡（决定 SDDM greeter 落点） ──
    # 优先级：1) 已知序列号  2) 横屏  3) 竖屏
    _find_primary_card() {
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
          _log "primary $_card ($_conn): matched serial 60PZCH3"
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
        _log "primary $_first_landscape: first landscape display"
        echo "$_first_landscape"; return 0
      fi
      if [ -n "$_first_portrait" ]; then
        _log "primary $_first_portrait: first portrait display (no landscape found)"
        echo "$_first_portrait"; return 0
      fi
      _log "no connected display found"
      return 1
    }

    # ── 列出所有 connected 的 DRM card（去重，alpha 顺序） ──
    # DisplayLink/evdi 每个 dongle 一张独立 card，必须全部喂给 kwin 才能扫描出口。
    _find_all_cards() {
      local _st _v _dir _conn _card _seen=" "
      for _st in /sys/class/drm/card*-*/status; do
        [ -f "$_st" ] || continue
        read -r _v < "$_st" 2>/dev/null
        [ "$_v" = "connected" ] || continue
        _dir=$(${dirname} "$_st")
        _conn=$(${basename} "$_dir")
        _card="''${_conn%%-*}"
        case "$_seen" in *" $_card "*) continue ;; esac
        _seen="$_seen$_card "
        echo "$_card"
      done
    }

    # ── 拼装 KWIN_DRM_DEVICES：intel-igpu(render) : primary : 其余 evdi cards ──
    _build_kwin_devices() {
      local _primary="$1" _all="$2" _devs="" _c
      [ -e /dev/dri/intel-igpu ] && _devs="/dev/dri/intel-igpu"
      if [ -n "$_primary" ]; then
        _devs="''${_devs:+$_devs:}/dev/dri/$_primary"
      fi
      for _c in $_all; do
        [ "$_c" = "$_primary" ] && continue
        _devs="''${_devs:+$_devs:}/dev/dri/$_c"
      done
      echo "$_devs"
    }

    # ── 1. 即时检测 ──
    _primary=$(_find_primary_card)
    _all=$(_find_all_cards | tr '\n' ' ')

    # ── 2. 无显示设备 → 事件驱动等待（非轮询） ──
    if [ -z "$_primary" ]; then
      _log "waiting for DRM device (event-driven)..."
      while IFS= read -r _; do
        sleep 1
        while IFS= read -r -t 0.5 _; do :; done
        _primary=$(_find_primary_card)
        _all=$(_find_all_cards | tr '\n' ' ')
        [ -n "$_primary" ] && break
      done < <(${udevadm} monitor -s drm -u)
    fi

    # ── 3. KWIN_DRM_DEVICES：iGPU 当 render，所有 connected card 当 scanout ──
    # 关键：永远用 udev symlink /dev/dri/intel-igpu 当 render node，不要赌 card 编号。
    # evdi 启动顺序不固定，有概率 card0 = evdi。evdi 没有 renderD*（虚拟扫描出口、
    # 没 GPU 单元），单独喂给 kwin 起不来 → SDDM "缩小动画" 后停在最后一帧。
    # 多块 DisplayLink dongle = 多张独立 evdi card，必须全部列出，否则 kwin 只看到一个。
    if [ -z "$_primary" ]; then
      _log "WARNING: no display found after event wait, letting kwin auto-detect"
    else
      KWIN_DRM_DEVICES=$(_build_kwin_devices "$_primary" "$_all")
      export KWIN_DRM_DEVICES
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

    # 监听 primary 或 connected card 集合的任何变化 → 重启 kwin 让
    # KWIN_DRM_DEVICES 重新枚举（hot-plug 第二张 evdi 也会触发）。
    _CUR_PRIMARY="$_primary"
    _CUR_ALL="$_all"
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
      _new_primary=$(_find_primary_card)
      _new_all=$(_find_all_cards | tr '\n' ' ')
      if [ "''${_new_primary:-}" != "$_CUR_PRIMARY" ] || [ "$_new_all" != "$_CUR_ALL" ]; then
        _log "display set changed: primary=$_CUR_PRIMARY all=[$_CUR_ALL] -> primary=''${_new_primary:-<none>} all=[$_new_all], restarting compositor"
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
