{ pkgs, unstable, ... }: {
  home.packages = with unstable; [
    brightnessctl
    cliphist
    dunst
    ffmpeg
    grim
    grimblast          # grim+slurp wrapper：区域/窗口/全屏截图一条命令搞定
    hypridle
    hyprlock
    hyprpolkitagent
    jq
    pkgs.kdePackages.kwallet  # Hyprland 会话也用 kwallet 做 Secret Service（跟随系统 KDE）
    lan-mouse            # 跨设备鼠标键盘共享（Software KVM）
    libnotify
    mpvpaper
    networkmanagerapplet # nm-connection-editor：图形化编辑 WiFi/VPN 配置
    pavucontrol
    playerctl
    rofi
    slurp
    socat
    swappy             # 截图标注编辑器
    quickshell
    swayimg
    swww
    waybar
    wf-recorder        # 轻量 Wayland 录屏，支持 slurp 区域选择
    wlr-randr
    wl-clipboard
  ];

  wayland.windowManager.hyprland = {
    enable = true;
    package = unstable.hyprland;
    # 系统侧已启用 programs.hyprland.withUWSM = true，UWSM 会自己管理
    # graphical-session.target / wayland-wm@hyprland.target。
    # Hyprland 官方 wiki 明确要求：使用 UWSM 时 HM 这边必须关掉 systemd 集成，
    # 否则与 UWSM 冲突。
    # https://github.com/hyprwm/hyprland-wiki/blob/main/content/Useful%20Utilities/Systemd-start.md
    systemd.enable = false;
    xwayland.enable = true;
    plugins = [
      unstable.hyprlandPlugins.hyprexpo
    ];

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
  xdg.configFile."hypr/scripts/video-wallpaper.sh" = {
    source = ./scripts/video-wallpaper.sh;
    executable = true;
  };
  xdg.configFile."hypr/scripts/power-menu.sh" = {
    source = ./scripts/power-menu.sh;
    executable = true;
  };
  xdg.configFile."hypr/base.conf".source = ./base.conf;
  xdg.configFile."hypr/keybinds.conf".source = ./keybinds.conf;
  xdg.configFile."hypr/hypridle.conf".source = ./hypridle.conf;
  xdg.configFile."hypr/hyprlock.conf".source = ./hyprlock.conf;
  xdg.configFile."hypr/monitors.conf".source = ./monitors.conf;
  xdg.configFile."hypr/windowrules.conf".source = ./windowrules.conf;
  xdg.configFile."uwsm/env-hyprland".source = ./env-hyprland;
  xdg.configFile."dunst/dunstrc".source = ./dunst/dunstrc;
  xdg.configFile."rofi/config.rasi".source = ./rofi/config.rasi;
  xdg.configFile."rofi/power-menu.rasi".source = ./rofi/power-menu.rasi;
  xdg.configFile."swayimg/config".text = ''
    [info]
    show = no
    [viewer]
    scale = fit
  '';
  xdg.configFile."waybar/config".source = ./waybar/config;
  xdg.configFile."waybar/style.css".source = ./waybar/style.css;
  xdg.configFile."quickshell" = { source = ./quickshell; recursive = true; };
}
