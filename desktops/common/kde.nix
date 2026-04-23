{self, lib, pkgs, unstable, ...} : {
  services.displayManager.sddm = {
    enable = true;
    wayland.enable = true;
    settings = {
      General = {
        GreeterEnvironment = "QT_FONT_DPI=128";
      };
      Theme = {
        CursorTheme = "breeze_cursors";
        CursorSize = 32;
      };
    };
  };

  environment.sessionVariables = {
    KWIN_DRM_PREFER_COLOR_DEPTH = "24";
    KWIN_DRM_NO_DIRECT_SCANOUT = "1";
  };

  # startplasma-wayland 会 source 该目录脚本；静态 displaylink-card 只匹配单张 evdi。
  environment.etc."xdg/plasma-workspace/env/10-kwin-drm-devices.sh" = {
    mode = "0755";
    text = ''
      _devs=""
      _intel_target=""
      if [ -e /dev/dri/intel-igpu ]; then
        _devs="/dev/dri/intel-igpu"
        if [ -L /dev/dri/intel-igpu ]; then
          _intel_target=$(basename "$(readlink -f /dev/dri/intel-igpu 2>/dev/null)" 2>/dev/null)
        fi
      fi
      _scanout=0
      _seen=" ''${_intel_target} "
      for _st in /sys/class/drm/card*-*/status; do
        [ -f "$_st" ] || continue
        read -r _v < "$_st" 2>/dev/null
        [ "$_v" = "connected" ] || continue
        _conn=$(basename "$(dirname "$_st")")
        _card="''${_conn%%-*}"
        case "$_seen" in *" $_card "*) continue ;; esac
        _seen="$_seen$_card "
        _devs="''${_devs:+$_devs:}/dev/dri/$_card"
        _scanout=$((_scanout + 1))
      done
      if [ "$_scanout" -gt 0 ]; then
        export KWIN_DRM_DEVICES="$_devs"
      fi
      unset _devs _intel_target _scanout _seen _st _v _conn _card
    '';
  };

  services.desktopManager.plasma6.enable = true;
}
