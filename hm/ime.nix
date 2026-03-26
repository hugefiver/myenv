{ unstable, ... }: {
  i18n.inputMethod = {
    enable = true;
    type = "fcitx5";
    fcitx5.waylandFrontend = true;
    fcitx5.addons = with unstable; [
      fcitx5-nord
      fcitx5-gtk
      kdePackages.fcitx5-qt
      libsForQt5.fcitx5-qt
      qt6Packages.fcitx5-chinese-addons
      fcitx5-fluent
      (fcitx5-rime.override {
        librime = unstable.librime.override {
          plugins = [ unstable.librime-lua ];
        };
        rimeDataPkgs = [
          unstable.rime-ice
        ];
      })
    ];
  };

  # Wayland 原生应用（KDE/GTK4/Qt6）通过 Wayland text-input-v3 协议
  # 直接与 fcitx5 Wayland frontend 通信，无需环境变量。
  # 以下变量为 XWayland 应用提供输入法支持：
  #   XMODIFIERS  → XIM 协议（所有 X11 应用的基础回退）
  #   GTK_IM_MODULE → GTK IM module（XWayland GTK 应用）
  #   QT_IM_MODULE  → Qt IM module（XWayland Qt 应用 + Hyprland 下的 Qt 应用）
  # fcitx5-gtk / fcitx5-qt 模块在 Wayland 会话中会自动使用 Wayland 协议，
  home.sessionVariables = {
    XMODIFIERS = "@im=fcitx";
    GTK_IM_MODULE = "fcitx";
    QT_IM_MODULE = "fcitx";
  };
}
