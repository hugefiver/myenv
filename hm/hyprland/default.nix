{ pkgs, ... }: {
  home.packages = with pkgs; [
    brightnessctl
    cliphist
    dunst
    grim
    grimblast          # grim+slurp wrapper：区域/窗口/全屏截图一条命令搞定
    hypridle
    hyprlock
    hyprpolkitagent
    jq
    kdePackages.kwallet  # Hyprland 会话也用 kwallet 做 Secret Service
    libnotify
    networkmanagerapplet # nm-connection-editor：图形化编辑 WiFi/VPN 配置
    pavucontrol
    playerctl
    rofi
    slurp
    socat
    swappy             # 截图标注编辑器
    swww
    waybar
    wf-recorder        # 轻量 Wayland 录屏，支持 slurp 区域选择
    wlogout
    wlr-randr
    wl-clipboard
  ];

  wayland.windowManager.hyprland = {
    enable = true;
    systemd.enable = true;
    xwayland.enable = true;

    extraConfig = ''
      source = ~/.config/hypr/monitors.conf
      source = ~/.config/hypr/base.conf
      source = ~/.config/hypr/windowrules.conf
      source = ~/.config/hypr/keybinds.conf
    '';
  };

  xdg.configFile."hypr/autostart.sh" = {
    source = ./autostart.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/record-toggle.sh" = {
    source = ./scripts/record-toggle.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/auto-scale.sh" = {
    source = ./scripts/auto-scale.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/keybind-cheatsheet.sh" = {
    source = ./scripts/keybind-cheatsheet.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/waybar-autohide.sh" = {
    source = ./scripts/waybar-autohide.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/rofi-launcher.sh" = {
    source = ./scripts/rofi-launcher.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/popup-dismiss.sh" = {
    source = ./scripts/popup-dismiss.sh;
    executable = true;
  };
  xdg.configFile."hypr/base.conf".source = ./config.txt;
  xdg.configFile."hypr/keybinds.conf".source = ./keybinds.conf;
  xdg.configFile."hypr/hypridle.conf".source = ./hypridle.conf;
  xdg.configFile."hypr/hyprlock.conf".source = ./hyprlock.conf;
  xdg.configFile."hypr/monitors.conf".source = ./monitors.conf;
  xdg.configFile."hypr/windowrules.conf".source = ./windowrules.conf;
  xdg.configFile."uwsm/env-hyprland".source = ./env-hyprland;
  xdg.configFile."dunst/dunstrc".source = ./dunst/dunstrc;
  xdg.configFile."rofi/config.rasi".source = ./rofi/config.rasi;
  xdg.configFile."waybar/config".source = ./waybar/config;
  xdg.configFile."waybar/style.css".source = ./waybar/style.css;
  xdg.configFile."wlogout/layout".source = ./wlogout/layout;
  xdg.configFile."wlogout/style.css".source = ./wlogout/style.css;
}
