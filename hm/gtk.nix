{ pkgs, unstable, config, ... }: {
  gtk = {
    enable = true;
    theme = {
      name = "Fluent-Dark";
      package = unstable.fluent-gtk-theme;
    };
    gtk4.theme = config.gtk.theme;
    iconTheme = {
      name = "Fluent-dark";
      package = unstable.fluent-icon-theme;
    };
    font = {
      name = "Noto Sans";
      size = 11;
    };
  };

  qt = {
    enable = true;
    platformTheme.name = "gtk";
  };

  home.packages = with pkgs; [
    adwaita-icon-theme
    hicolor-icon-theme
    kdePackages.breeze-icons
  ];
}
