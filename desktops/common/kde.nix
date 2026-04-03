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

  # 仅在用户会话中设置多 GPU 变量，避免 SDDM 的 kwin_wayland 继承后
  # 尝试在 DisplayLink 输出上渲染导致卡顿。
  environment.sessionVariables = {
    KWIN_DRM_PREFER_COLOR_DEPTH = "24";
    KWIN_DRM_DEVICES = "/dev/dri/intel-igpu:/dev/dri/displaylink-card";
    KWIN_DRM_NO_DIRECT_SCANOUT = "1";
  };

  services.desktopManager.plasma6.enable = true;
}
