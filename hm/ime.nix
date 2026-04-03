{ unstable, lib, ... }:
let
  fcitx5-rime-pkg = unstable.fcitx5-rime.override {
    librime = unstable.librime.override {
      plugins = [ unstable.librime-lua ];
    };
    rimeDataPkgs = [
      unstable.rime-ice
    ];
  };
in {
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
      fcitx5-rime-pkg
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

  xdg.configFile."fcitx5/config".text = ''
    [Behavior]
    ShareInputState=No
  '';

  home.activation.rimeDeploySchemas = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    RIME_DIR="''${XDG_DATA_HOME:-$HOME/.local/share}/fcitx5/rime"
    mkdir -p "$RIME_DIR"
    run ${fcitx5-rime-pkg}/bin/rime_deployer --build "$RIME_DIR" 2>/dev/null || true
  '';

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

  # ── RIME 全局配置 ──────────────────────────────────────────────
  xdg.dataFile."fcitx5/rime/default.custom.yaml".text = ''
    patch:
      schema_list:
        - schema: rime_ice
      menu:
        page_size: 7
        alternative_select_labels: [ ①, ②, ③, ④, ⑤, ⑥, ⑦, ⑧, ⑨, ⑩ ]
      ascii_composer:
        switch_key:
          Caps_Lock: clear
          Shift_L: commit_code
          Shift_R: commit_code
          Control_L: noop
          Control_R: noop
  '';

  # ── RIME：rime_ice schema 个性化 ────────────────────────────────
  xdg.dataFile."fcitx5/rime/rime_ice.custom.yaml".text = ''
    patch:
      # 默认英文，左/右 Shift 切换
      "switches/@0/states": [ Ａ, 中 ]
      "switches/@0/reset": 1

      # 模糊音
      "speller/algebra/@before 0":
        - derive/^([zcs])h/$1/          # zh ch sh → z c s
        - derive/^([zcs])([^h])/$1h$2/  # z c s → zh ch sh
        - derive/^l/n/
        - derive/^n/l/
        - derive/an$/ang/
        - derive/en$/eng/
        - derive/in$/ing/
  '';
}
