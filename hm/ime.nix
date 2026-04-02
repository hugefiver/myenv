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

  # 环境变量策略（参考 https://fcitx-im.org/wiki/Using_Fcitx_5_on_Wayland）
  #
  # XMODIFIERS：全局设置，XWayland 应用通过 XIM 协议连接 fcitx5。
  #
  # GTK_IM_MODULE / QT_IM_MODULE：不在此全局设置！
  #   KDE Plasma：原生支持 text-input-v2/v3，全局设置会导致候选窗闪烁。
  #   Hyprland：在 hyprland/config.txt 的 env 中单独设置
  #            （Qt 无 text-input-v2 支持，必须用 fcitx IM module）。
  home.sessionVariables = {
    XMODIFIERS = "@im=fcitx";
  };

  # ── fcitx5 profile：键盘 + RIME ──────────────────────────────
  xdg.configFile."fcitx5/profile" = {
    force = true;
    text = ''
      [Groups/0]
      Name=Default
      Default Layout=us
      DefaultIM=rime

      [Groups/0/Items/0]
      Name=keyboard-us
      Layout=

      [Groups/0/Items/1]
      Name=rime
      Layout=

      [GroupOrder]
      0=Default
    '';
  };

  # ── RIME：确保使用雾凇拼音 schema ──────────────────────────────
  # rime-ice 的 default.yaml 已在共享数据目录（rimeDataPkgs），
  # 此 custom 文件覆盖用户目录中可能存在的旧配置。
  xdg.dataFile."fcitx5/rime/default.custom.yaml".text = ''
    patch:
      schema_list:
        - schema: rime_ice
  '';
}
