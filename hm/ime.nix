{config, pkgs, unstable, ...} : {
  i18n.inputMethod = {
    enable = true;
    type = "fcitx5";
    fcitx5.waylandFrontend = true;
    fcitx5.addons = with unstable; [ 
      fcitx5-nord fcitx5-gtk qt6Packages.fcitx5-chinese-addons libsForQt5.fcitx5-qt 
      fcitx5-fluent
      (fcitx5-rime.override {
        rimeDataPkgs = [
          unstable.rime-ice
        ];
      })
    ];
  };
}
