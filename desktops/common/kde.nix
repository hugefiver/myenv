{self, lib, pkgs, unstable, ...} : {
  services.displayManager.sddm = {
    enable = true;
    wayland.enable = true;
    settings = {
      Theme = {
        CursorTheme = "breeze_cursors";
        CursorSize = 24;
      };
    };
  };

  environment.variables = {
    KWIN_DRM_PREFER_COLOR_DEPTH = "24";
    KWIN_DRM_DEVICES = "/dev/dri/intel-igpu:/dev/dri/displaylink-card";
    KWIN_DRM_NO_DIRECT_SCANOUT = "1";
  };

  services.desktopManager.plasma6.enable = true;
}
