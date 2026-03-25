# from https://github.com/YuJuncen/nix-configuration/blob/master/home-manager/hidpi.nix

{ ... }: {
  home.sessionVariables = {
    _JAVA_OPTIONS = "-Dsun.java2d.uiScale=2";
    QT_AUTO_SCREEN_SCALE_FACTOR = 1;
    QT_ENABLE_HIGHDPI_SCALING = 1;
  };
  xresources.properties = {
    "Xft.dpi" = 192;
    "Xft.autohint" = 0;
    "Xft.lcdfilter" = "lcddefault";
    "Xft.hintstyle" = "hintfull";
    "Xft.hinting" = 1;
    "Xft.antialias" = 1;
    "Xft.rgba" = "rgb";
  };
  services.xsettingsd = {
    enable = true;
    settings = {
      "Gdk/WindowScalingFactor" = 2;
      "Gdk/UnscaledDPI" = 98340;
    };
  };
}