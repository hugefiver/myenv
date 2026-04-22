{ unstable, lib, ... }:
let
  # nixpkgs 的 librime 默认 plugins = [ librime-lua librime-octagram ]，
  # 直接用即可走二进制缓存。早期这里做 .override { plugins = [ librime-lua ]; }
  # 反而触发本地编译。
  custom-librime = unstable.librime;
  fcitx5-rime-pkg = unstable.fcitx5-rime.override {
    librime = custom-librime;
    rimeDataPkgs = [
      unstable.rime-ice
    ];
  };
in {
  i18n.inputMethod = {
    enable = true;
    type = "fcitx5";
    fcitx5.fcitx5-with-addons = unstable.qt6Packages.fcitx5-with-addons;
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
  #   Hyprland：在 hyprland/base.conf 的 env 中单独设置
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

    # 调试：验证 store 路径
    echo "rime_deployer: ${custom-librime}/bin/rime_deployer"
    ls -la ${custom-librime}/bin/rime_deployer 2>&1 || echo "WARNING: rime_deployer not found!"
    echo "rime-data dir: ${fcitx5-rime-pkg}/share/rime-data"
    ls ${fcitx5-rime-pkg}/share/rime-data/ 2>&1 | head -20 || echo "WARNING: rime-data dir empty or missing!"

    # rime_deployer --build 使用位置参数： user_data_dir [shared_data_dir]
    # 不是 --shared-data-dir flag！之前误用 flag 导致构建失败。
    run ${custom-librime}/bin/rime_deployer --build "$RIME_DIR" ${fcitx5-rime-pkg}/share/rime-data || echo "WARNING: rime_deployer failed with exit code $?"
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
      Name=rime
      Layout=

      [Groups/0/Items/1]
      Name=keyboard-us
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
