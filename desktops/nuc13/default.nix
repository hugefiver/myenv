{
  self,
  config,
  inputs,
  lib,
  pkgs,
  unstable,
  system,
  ...
} : let
  mihomotuiRepo = "/home/hugefiver/.local/share/mihomotui";
  legacyConfig = "/home/hugefiver/.local/share/io.github.clash-verge-rev.clash-verge-rev/clash-verge.yaml";
  migrationId = "6d69686f6d6f7475692d6d6967726174";
  syncMihomotuiText = ''
    set -eu

    if [ "$#" -ne 2 ] && [ "$#" -ne 3 ]; then
      printf '%s\n' 'usage: mihomotui-sync-profile-repo REPO LEGACY [--emit-active]' >&2
      exit 64
    fi
    if [ "$#" -eq 3 ] && [ "$3" != "--emit-active" ]; then
      printf '%s\n' 'usage: mihomotui-sync-profile-repo REPO LEGACY [--emit-active]' >&2
      exit 64
    fi

    REPO="$1"
    LEGACY="$2"
    EMIT_ACTIVE="''${3:-}"
    MIGRATION_ID='${migrationId}'
    MIGRATION_FILE="profiles/$MIGRATION_ID.yaml"
    MIGRATION_PAYLOAD="$REPO/$MIGRATION_FILE"
    INDEX="$REPO/profiles.json"
    ACTIVE="$REPO/active.yaml"
    seed_stage=""
    seed_snapshot=""
    snapshot=""
    migration_stage=""

    umask 077

    die() {
      printf '%s\n' "mihomotui-sync-profile-repo: $*" >&2
      exit 1
    }

    is_regular_file() {
      [ ! -L "$1" ] && [ -f "$1" ]
    }

    emit_active() {
      if [ "$EMIT_ACTIVE" = "--emit-active" ]; then
        if [ "$ACTIVE_ID" = "$MIGRATION_ID" ]; then
          cat -- "$MIGRATION_PAYLOAD"
        else
          cat -- "$ACTIVE"
        fi
      fi
    }

    cleanup() {
      for temporary in "$seed_stage" "$seed_snapshot" "$snapshot" "$migration_stage"; do
        if [ -n "$temporary" ]; then
          rm -rf -- "$temporary"
        fi
      done
    }
    trap cleanup EXIT

    if ! is_regular_file "$LEGACY"; then
      die "legacy config must be a regular, non-symlink file: $LEGACY"
    fi

    if [ -L "$REPO" ]; then
      die "profile repository must not be a symlink: $REPO"
    fi

    if [ ! -e "$REPO" ]; then
      repo_parent=$(dirname "$REPO")
      install -d -m 0755 "$repo_parent"
      seed_stage=$(mktemp -d "$repo_parent/.mihomotui-seed.XXXXXX") || die "failed to create seed stage"
      chmod 0700 "$seed_stage"
      mkdir -m 0700 "$seed_stage/profiles"
      seed_snapshot=$(mktemp "$repo_parent/.mihomotui-legacy.XXXXXX") || die "failed to create legacy snapshot"
      cp -- "$LEGACY" "$seed_snapshot"
      chmod 0600 "$seed_snapshot"
      cp -- "$seed_snapshot" "$seed_stage/$MIGRATION_FILE"
      cp -- "$seed_snapshot" "$seed_stage/active.yaml"
      timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
      printf '{\n  "version": 1,\n  "active_id": "%s",\n  "profiles": [\n    {\n      "id": "%s",\n      "name": "clash-verge-migration",\n      "kind": "file",\n      "file": "profiles/%s.yaml",\n      "created_at": "%s",\n      "updated_at": "%s"\n    }\n  ]\n}\n' \
        "$MIGRATION_ID" "$MIGRATION_ID" "$MIGRATION_ID" "$timestamp" "$timestamp" > "$seed_stage/profiles.json"
      chmod 0600 "$seed_stage/$MIGRATION_FILE" "$seed_stage/active.yaml" "$seed_stage/profiles.json"

      if mv -T --no-clobber -- "$seed_stage" "$REPO" && [ ! -e "$seed_stage" ]; then
        seed_stage=""
        ACTIVE_ID="$MIGRATION_ID"
        emit_active
        exit 0
      fi
      if [ ! -e "$REPO" ] && [ ! -L "$REPO" ]; then
        die "failed to install seeded profile repository"
      fi
    fi

    if [ -L "$REPO" ]; then
      die "profile repository must not be a symlink: $REPO"
    fi
    if [ ! -d "$REPO" ]; then
      die "profile repository must be a directory: $REPO"
    fi
    if [ -L "$REPO/profiles" ] || [ ! -d "$REPO/profiles" ]; then
      die "profiles must be a non-symlink directory: $REPO/profiles"
    fi
    if ! is_regular_file "$INDEX"; then
      die "profile index must be a regular, non-symlink file: $INDEX"
    fi
    if ! is_regular_file "$MIGRATION_PAYLOAD"; then
      die "migration payload must be a regular, non-symlink file: $MIGRATION_PAYLOAD"
    fi
    if ! is_regular_file "$ACTIVE"; then
      die "active config must be a regular, non-symlink file: $ACTIVE"
    fi
    ACTIVE_ID=$(jq -er --arg id "$MIGRATION_ID" '
      if (
        (.version == 1) and
        (.active_id | type == "string") and
        (.profiles | type == "array") and
        ([.profiles[] | select(type == "object" and .id == $id)] | length == 1) and
        ([.profiles[] | select(
          type == "object" and
          .id == $id and
          .name == "clash-verge-migration" and
          .kind == "file" and
          .file == ("profiles/" + $id + ".yaml") and
          ((has("url") | not) or .url == "")
        )] | length == 1)
      ) then
        .active_id
      else
        error("invalid migration metadata")
      end
    ' "$INDEX") || die "profile index has an unexpected migration entry: $INDEX"

    snapshot=$(mktemp "$REPO/.mihomotui-legacy.XXXXXX") || die "failed to create legacy snapshot"
    cp -- "$LEGACY" "$snapshot"
    chmod 0600 "$snapshot"
    migration_stage=$(mktemp "$REPO/profiles/.mihomotui-migration.XXXXXX") || die "failed to create migration stage"
    cp -- "$snapshot" "$migration_stage"
    chmod 0600 "$migration_stage"

    if ! mv -T -- "$migration_stage" "$MIGRATION_PAYLOAD"; then
      die "failed to install migration payload"
    fi
    migration_stage=""
    emit_active
  '';
  syncMihomotui = pkgs.writeShellApplication {
    name = "mihomotui-sync-profile-repo";
    runtimeInputs = [ pkgs.coreutils pkgs.jq ];
    text = syncMihomotuiText;
  };
  mihomotuiTestInterceptors = pkgs.runCommand "mihomotui-sync-test-interceptors" {} ''
    mkdir -p "$out/bin"
    cat > "$out/bin/cp" <<'EOF'
    #!${pkgs.runtimeShell}
    set -eu
    "${pkgs.coreutils}/bin/cp" "$@"
    if [ "''${MIHOMOTUI_TEST_MUTATE_LEGACY:-}" = "1" ] && [ "$#" -eq 3 ] && [ "$1" = "--" ]; then
      source_path="$2"
      destination_path="$3"
      case "$destination_path" in
        */.mihomotui-legacy.*)
          printf '%s\n' 'mihomotui test mutation' > "$source_path"
          ;;
      esac
    fi
    EOF
    cat > "$out/bin/mv" <<'EOF'
    #!${pkgs.runtimeShell}
    set -eu
    source_path=""
    destination_path=""
    for argument in "$@"; do
      source_path="$destination_path"
      destination_path="$argument"
    done
    if [ "''${MIHOMOTUI_TEST_CONTEST_SEED:-}" = "1" ]; then
      case "$source_path" in
        */.mihomotui-seed.*)
          "${pkgs.coreutils}/bin/rm" -rf -- "$destination_path"
          "${pkgs.coreutils}/bin/mkdir" -- "$destination_path"
          printf '%s\n' 'mihomotui test interceptor: contesting seed destination' >&2
          exit 1
          ;;
      esac
    fi
    exec "${pkgs.coreutils}/bin/mv" "$@"
    EOF
    chmod 0555 "$out/bin/cp" "$out/bin/mv"
  '';
  syncMihomotuiTest = pkgs.writeShellApplication {
    name = "mihomotui-sync-profile-repo-test";
    runtimeInputs = [ mihomotuiTestInterceptors pkgs.coreutils pkgs.jq ];
    text = syncMihomotuiText;
  };
  mihomotuiSyncCommand = lib.escapeShellArgs [
    "${pkgs.util-linux}/bin/runuser"
    "-u"
    "hugefiver"
    "--"
    "${syncMihomotui}/bin/mihomotui-sync-profile-repo"
    mihomotuiRepo
    legacyConfig
  ];
  mihomotuiEmitActiveCommand = lib.escapeShellArgs [
    "${pkgs.util-linux}/bin/runuser"
    "-u"
    "hugefiver"
    "--"
    "${syncMihomotui}/bin/mihomotui-sync-profile-repo"
    mihomotuiRepo
    legacyConfig
    "--emit-active"
  ];
  prepareMihomoBoot = pkgs.writeShellApplication {
    name = "mihomo-boot-prepare";
    runtimeInputs = [ pkgs.coreutils ];
    text = ''
      set -eu

      RUN_DIR="''${RUNTIME_DIRECTORY:-/run/mihomo-boot}"
      temporary=""

      cleanup() {
        if [ -n "$temporary" ]; then
          rm -f -- "$temporary"
        fi
      }
      trap cleanup EXIT

      temporary=$(mktemp "$RUN_DIR/.mihomotui-source.XXXXXX")
      chmod 0600 "$temporary"
      if ! ${mihomotuiEmitActiveCommand} > "$temporary"; then
        exit 1
      fi
      mv -fT -- "$temporary" "$RUN_DIR/source.yaml"
      temporary=""
    '';
  };
in {
  imports = [
    ../common
    ../common/personal.nix
    ../common/hyprland.nix
    ../common/kde.nix

    ./disk.nix
  ];

  boot.kernelParams = ["default_hugepagesz=2M" "hugepagesz=1G" "hugepages=2"];
  boot.kernel.sysctl = {
    "vm.nr_hugepages" = 1260;
  };

  boot.kernel.sysctl."net.ipv4.tcp_congestion_control" = "bbr";
  boot.kernel.sysctl."net.core.rmem_max" = 16777216;
  boot.kernel.sysctl."net.core.wmem_max" = 16777216;
  boot.kernel.sysctl."net.ipv4.tcp_rmem" = "4096 87380 16777216";
  boot.kernel.sysctl."net.ipv4.tcp_wmem" = "4096 87380 16777216";

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
  networking.firewall = {
    trustedInterfaces = [ "Meta" "Mihomo" "MihomoBoot" ];
    extraReversePathFilterRules = ''
      iifname { "Meta", "Mihomo", "MihomoBoot" } accept comment "mihomo TUN"
    '';
    allowedUDPPorts = [ 4242 ];  # lan-mouse
  };

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

  system.activationScripts.mihomotuiProfileSeed = lib.stringAfter [ "users" ] ''
    ${mihomotuiSyncCommand}
  '';
  system.build.mihomotuiProfileSeed = syncMihomotui;
  system.build.mihomotuiProfileSyncTest = syncMihomotuiTest;

  # SSH 期透明代理；登录桌面后 mihomo-boot-handoff 同步停止，verge 接管
  systemd.services.mihomo-boot =
    let
      bootHome = "/var/lib/mihomo-boot";
      geoipData = "${pkgs.v2ray-geoip}/share/v2ray/geoip.dat";
      countryMMDB = pkgs.dbip-country-lite.mmdb;
      geositeData = pkgs.fetchurl {
        url = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/download/202607092306/geosite.dat";
        hash = "sha256-E1Vs8u0vkDtgXQNHJejL0Xenl/fYn5IE9npD2M0Js8o=";
      };
      bootDevice = "MihomoBoot";
      bootTable = "12022";
      bootRule = "9200";
      startScript = pkgs.writeShellApplication {
        name = "mihomo-boot-start";
        runtimeInputs = [ unstable.mihomo pkgs.coreutils pkgs.iproute2 pkgs.nftables pkgs.procps pkgs.gnugrep pkgs.yq-go ];
        text = ''
          BOOT_HOME='${bootHome}'
          RUN_DIR="''${RUNTIME_DIRECTORY:-/run/mihomo-boot}"
          RUN_CFG="$RUN_DIR/config.yaml"
          SOURCE_CFG="$RUN_DIR/source.yaml"
          install -m 0644 '${geoipData}' "$BOOT_HOME/geoip.dat"
          install -m 0644 '${countryMMDB}' "$BOOT_HOME/Country.mmdb"
          install -m 0644 '${geositeData}' "$BOOT_HOME/GeoSite.dat"

          # 等非-TUN default 路由（最多 60s）
          for i in $(seq 1 60); do
            if ip -4 route show default 2>/dev/null \
                | grep -E '^default ' \
                | grep -vE 'dev (Meta|Mihomo|${bootDevice})' \
                | grep -q .; then
              echo "[mihomo-boot] default route ready after ''${i}s"
              break
            fi
            sleep 1
          done

          LAN_EXCLUDES=$(ip -4 route show scope link 2>/dev/null \
            | grep -vE 'dev (Meta|Mihomo|${bootDevice})' \
            | while read -r dst _; do
                case "$dst" in
                  default|198.18.*) ;;
                  *) printf '%s,' "$dst" ;;
                esac
              done)
          ROUTE_EXCLUDES='[192.168.0.0/16,10.0.0.0/8,172.16.0.0/12,169.254.0.0/16,100.64.0.0/10,fc00::/7,fe80::/10]'
          if [ -n "$LAN_EXCLUDES" ]; then
            ROUTE_EXCLUDES="[''${LAN_EXCLUDES%,},192.168.0.0/16,10.0.0.0/8,172.16.0.0/12,169.254.0.0/16,100.64.0.0/10,fc00::/7,fe80::/10]"
          fi
          export ROUTE_EXCLUDES

          for i in $(seq 1 15); do
            if ! pgrep -f 'verge-mihomo' >/dev/null 2>&1; then
              break
            fi
            if [ "$i" -eq 15 ]; then
              echo "[mihomo-boot] clash-verge core still running; skip boot core"
              exit 0
            fi
            sleep 1
          done

          if ip link show ${bootDevice} >/dev/null 2>&1 || ip link show Mihomo >/dev/null 2>&1 || ip link show Meta >/dev/null 2>&1; then
            ip link del ${bootDevice} 2>/dev/null || true
            for p in ${bootRule} $(( ${bootRule} + 1 )) $(( ${bootRule} + 2 )) $(( ${bootRule} + 10 )); do
              ip rule del priority "$p" 2>/dev/null || true
            done
            ip route flush table ${bootTable} 2>/dev/null || true
          fi

          yq '
            .["geodata-mode"] = true |
            .tun.enable = true |
            .tun.device = "${bootDevice}" |
            .tun.stack = "gvisor" |
            .tun.auto-route = true |
            .tun.auto-redirect = true |
            .tun.auto-detect-interface = true |
            .tun.dns-hijack = ["any:53"] |
            .tun.strict-route = false |
            .tun.iproute2-table-index = ${bootTable} |
            .tun.iproute2-rule-index = ${bootRule} |
            .tun.route-exclude-address = (strenv(ROUTE_EXCLUDES) | from_yaml) |
            .dns.enable = true |
            .dns.listen = "127.0.0.1:8854" |
            .dns.enhanced-mode = "fake-ip" |
            .dns.fake-ip-range = "198.18.0.1/16" |
            .port = 0 |
            .socks-port = 0 |
            .mixed-port = 0 |
            .redir-port = 0 |
            .tproxy-port = 0 |
            .external-controller = "127.0.0.1:9090" |
            .external-controller-unix = "" |
            .secret = "" |
            del(.external-controller-pipe) |
            del(.external-controller-tls)
          ' "$SOURCE_CFG" > "$RUN_CFG"

          mihomo -t -d "$BOOT_HOME" -f "$RUN_CFG"
          exec mihomo -d "$BOOT_HOME" -f "$RUN_CFG"
        '';
      };
      # SIGKILL/崩溃兜底；正常 SIGTERM mihomo 自清
      teardown = pkgs.writeShellApplication {
        name = "mihomo-boot-teardown";
        runtimeInputs = [ pkgs.iproute2 ];
        text = ''
          ip link del ${bootDevice} 2>/dev/null || true
          for p in ${bootRule} $(( ${bootRule} + 1 )) $(( ${bootRule} + 2 )) $(( ${bootRule} + 10 )); do
            ip rule del priority "$p" 2>/dev/null || true
          done
          ip route flush table ${bootTable} 2>/dev/null || true
        '';
      };
    in
    {
      description = "mihomo (boot-time, handed off to clash-verge GUI after login)";
      wantedBy = [ "multi-user.target" ];
      unitConfig = {
        StartLimitIntervalSec = 0;
      };
      serviceConfig = {
        Type = "simple";
        ExecStartPre = "${prepareMihomoBoot}/bin/mihomo-boot-prepare";
        ExecStart = "${startScript}/bin/mihomo-boot-start";
        ExecStopPost = "${teardown}/bin/mihomo-boot-teardown";
        Restart = "no";
        TimeoutStartSec = "90s";
        RuntimeDirectory = "mihomo-boot";
        RuntimeDirectoryMode = "0700";
        StateDirectory = "mihomo-boot";
        StateDirectoryMode = "0700";
      };
    };
  boot.kernelPackages = unstable.linuxPackages_zen;
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

  # netbird daemon；登录与配置由用户手动 `netbird up` / `netbird login` 管理
  services.netbird = {
    enable = true;
    package = unstable.netbird;
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

  services.displayManager.sddm.theme = "breeze-blank";
  environment.systemPackages = [
    (pkgs.stdenv.mkDerivation {
      pname = "sddm-theme-breeze-blank";
      version = "1.0";
      src = "${pkgs.kdePackages.plasma-desktop}/share/sddm/themes/breeze";
      installPhase = ''
        mkdir -p $out/share/sddm/themes/breeze-blank
        cp -r * $out/share/sddm/themes/breeze-blank/
        chmod -R u+w $out/share/sddm/themes/breeze-blank

        cd $out/share/sddm/themes/breeze-blank
        sed -i "s/^Name=.*/Name=breeze-blank/" metadata.desktop
        substituteInPlace Main.qml \
          --replace-fail '        onPressed: uiVisible = true;
        onPositionChanged: uiVisible = true;' '        onPressed: resetBlanking()
        onPositionChanged: resetBlanking()' \
          --replace-fail '        Keys.onPressed: event => {
            uiVisible = true;
            event.accepted = false;
        }' '        Keys.onPressed: event => {
            resetBlanking()
            event.accepted = false
        }'

        # Append root-level declarations inside the upstream Breeze Item.
        sed -i "$ d" Main.qml
        cat >> Main.qml <<'QML'

    function resetBlanking() {
        loginScreenRoot.uiVisible = true
        if (blackOverlay.visible) {
            blackOverlay.visible = false
            userListComponent.mainPasswordBox.forceActiveFocus()
        }
        blankTimer.restart()
    }

    Timer {
        id: blankTimer
        interval: 60000
        running: true
        repeat: false
        onTriggered: {
            blackOverlay.visible = true
            blackOverlay.forceActiveFocus()
        }
    }

    Rectangle {
        id: blackOverlay
        anchors.fill: parent
        color: "black"
        visible: false
        z: 99999
        focus: visible

        onVisibleChanged: {
            if (visible) {
                forceActiveFocus()
            }
        }

        Keys.onPressed: event => {
            resetBlanking()
            event.accepted = true
        }

        MouseArea {
            anchors.fill: parent
            hoverEnabled: true
            acceptedButtons: Qt.AllButtons
            onEntered: resetBlanking()
            onPositionChanged: resetBlanking()
            onPressed: mouse => {
                resetBlanking()
                mouse.accepted = true
            }
            onWheel: wheel => {
                resetBlanking()
                wheel.accepted = true
            }
        }
    }



    Connections {
        target: userListComponent.mainPasswordBox
        ignoreUnknownSignals: true
        function onTextChanged() { resetBlanking() }
        function onCursorPositionChanged() { resetBlanking() }
    }
}
QML
      '';
    })
  ];
  services.displayManager.sddm.settings.Wayland.CompositorCommand = let
    kwin = lib.getExe' pkgs.kdePackages.kwin "kwin_wayland";
    dirname = "${pkgs.coreutils}/bin/dirname";
    basename = "${pkgs.coreutils}/bin/basename";
    grep = "${pkgs.gnugrep}/bin/grep";
    od = "${pkgs.coreutils}/bin/od";
    mkfifo = "${pkgs.coreutils}/bin/mkfifo";
    mktemp = "${pkgs.coreutils}/bin/mktemp";
    rm = "${pkgs.coreutils}/bin/rm";
    udevadm = "${pkgs.systemd}/bin/udevadm";
    pgrep = "${pkgs.procps}/bin/pgrep";
    jq = "${pkgs.jq}/bin/jq";
    seq = "${pkgs.coreutils}/bin/seq";
    kscreenDoctor = lib.getExe' pkgs.kdePackages.libkscreen "kscreen-doctor";
    swayidle = "${pkgs.swayidle}/bin/swayidle";
  in toString (pkgs.writeShellScript "sddm-compositor" ''
    export KWIN_DRM_NO_DIRECT_SCANOUT=1
    _log() { echo "[sddm-compositor] $*" >&2; }
    _RUNTIME_DIR="''${XDG_RUNTIME_DIR:-/var/run/sddm}"
    export XDG_RUNTIME_DIR="$_RUNTIME_DIR"
    export WAYLAND_DISPLAY="''${WAYLAND_DISPLAY:-wayland-0}"
    _FIFO_DIR=$(${mktemp} -d "$_RUNTIME_DIR/sddm-drm-monitor.XXXXXX") || { _log "ERROR: failed to create DRM monitor directory in $_RUNTIME_DIR"; exit 1; }
    _FIFO="$_FIFO_DIR/events"
    _KWIN_PID=""
    _UDEV_PID=""
    _KSCREEN_PID=""
    _IDLE_PID=""
    _cleanup() {
      trap - EXIT
      [ -n "''${_IDLE_PID:-}" ] && kill "$_IDLE_PID" 2>/dev/null
      [ -n "''${_KSCREEN_PID:-}" ] && kill "$_KSCREEN_PID" 2>/dev/null
      [ -n "''${_KWIN_PID:-}" ] && ${kscreenDoctor} --dpms on >/dev/null 2>&1
      [ -n "''${_UDEV_PID:-}" ] && kill "$_UDEV_PID" 2>/dev/null
      [ -n "''${_KWIN_PID:-}" ] && kill "$_KWIN_PID" 2>/dev/null
      [ -n "''${_FIFO_DIR:-}" ] && ${rm} -rf "$_FIFO_DIR"
    }
    trap '_cleanup' EXIT

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
    ${mkfifo} "$_FIFO" || { _log "ERROR: failed to create DRM monitor FIFO"; exit 1; }
    ${udevadm} monitor -s drm -u > "$_FIFO" 2>/dev/null &
    _UDEV_PID=$!

    trap '_cleanup; wait "$_KWIN_PID" 2>/dev/null; exit' TERM INT HUP

    # ── 4.5 SDDM greeter idle blanking: DPMS off after 60s, on at activity ──
    (
      for _try in $(${seq} 1 40); do
        [ -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ] && break
        sleep 0.5
      done

      if [ ! -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ]; then
        _log "WARNING: Wayland socket not ready, skipping greeter idle DPMS"
        exit 0
      fi

      ${kscreenDoctor} --dpms on >/dev/null 2>&1 || _log "WARNING: failed to force greeter DPMS on"
      _log "starting 60s greeter idle DPMS"
      exec ${swayidle} -w \
        timeout 60 '${kscreenDoctor} --dpms off' \
        resume '${kscreenDoctor} --dpms on' \
        before-sleep '${kscreenDoctor} --dpms on'
    ) &
    _IDLE_PID=$!

    # ── 4.6 Force DisplayLink greeter output to 1080p30 when KWin is ready ──
    (
      _doctor="${kscreenDoctor}"
      _set_any=0
      for _try in $(${seq} 1 40); do
        if [ ! -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ]; then
          sleep 0.5
          continue
        fi

        _json=$($_doctor --json 2>/dev/null) || { sleep 0.5; continue; }
        _outputs=$(printf '%s\n' "$_json" | ${jq} -r '.outputs[]? | select(.connected == true) | .id' 2>/dev/null)
        [ -n "$_outputs" ] || { sleep 0.5; continue; }

        _set_any=0
        for _out in $_outputs; do
          case "$_out" in
            ""|*[!0-9]*)
              _log "WARNING: skipping unexpected kscreen output id: $_out"
              continue
              ;;
          esac

          if $_doctor "output.$_out.mode.1920x1080@30" >/dev/null 2>&1; then
            _log "set output $_out to 1920x1080@30"
            _set_any=1
          fi
        done

        [ "$_set_any" -eq 1 ] && break
        sleep 0.5
      done
      [ "$_set_any" -eq 1 ] || _log "WARNING: could not set any output to 1920x1080@30"
    ) &
    _KSCREEN_PID=$!
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
