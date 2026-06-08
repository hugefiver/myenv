{ pkgs, unstable, ... }: let
  wayleConfig = (pkgs.formats.toml {}).generate "wayle-config.toml" {
    general = {
      "font-sans" = "Noto Sans";
      "font-mono" = "CaskaydiaCove Nerd Font Mono";
    };

    bar = {
      location = "top";
      scale = 0.9;
      "inset-edge" = 2.0;
      "inset-ends" = 6.0;
      padding = 0.0;
      "padding-ends" = 0.0;
      "module-gap" = 0.25;
      bg = "bg-surface";
      "background-opacity" = 0;
      "border-location" = "none";
      rounding = "none";
      shadow = "none";
      "button-variant" = "block-prefix";
      "button-bg-opacity" = 78;
      "button-icon-size" = 0.9;
      "button-label-size" = 0.9;
      "button-label-weight" = "semibold";
      "button-rounding" = "full";
      "button-border-location" = "all";
      "button-border-width" = 1;
      "button-group-background" = "bg-surface";
      "button-group-opacity" = 78;
      "button-group-border-color" = "border-accent";
      "button-group-rounding" = "full";
      "dropdown-shadow" = true;
      "dropdown-opacity" = 100;
      "dropdown-autohide" = true;
      "dropdown-freeze-label" = true;

      layout = [
        {
          monitor = "DVI-I-1";
          show = true;
          left = [
            "custom-launcher"
            "hyprland-workspaces"
            "window-title"
          ];
          center = [
            "media"
            "clock"
          ];
          right = [
            { name = "hardware"; modules = [ "cpu" "ram" "custom-temperature" ]; }
            { name = "status"; modules = [ "volume" "network" "battery" ]; }
            "bluetooth"
            "systray"
            "custom-power"
          ];
        }
        {
          monitor = "DVI-I-2";
          show = true;
          left = [ "hyprland-workspaces" ];
          center = [];
          right = [];
        }
      ];
    };

    styling = {
      scale = 1.0;
      rounding = "sm";
      "theme-provider" = "wayle";
      palette = {
        bg = "#141420";
        surface = "#2a2520";
        elevated = "#332d26";
        fg = "#e6ddd4";
        "fg-muted" = "#968b80";
        primary = "#dba86b";
        red = "#f28b82";
        yellow = "#dba86b";
        green = "#98bb6c";
        blue = "#7fb4ca";
      };
    };

    modules = {
      custom = [
        {
          id = "launcher";
          "icon-name" = "tb-grid-dots-symbolic";
          "icon-show" = true;
          "label-show" = false;
          "icon-bg-color" = "yellow";
          "left-click" = "~/.config/hypr/scripts/rofi-launcher.sh";
        }
        {
          id = "temperature";
          command = ''
            for f in /sys/devices/platform/coretemp.0/hwmon/hwmon*/temp1_input; do
              [ -r "$f" ] && awk '{printf "%d°C\\n", $1 / 1000}' "$f" && exit
            done
            echo "--°C"
          '';
          "interval-ms" = 5000;
          "icon-name" = "ld-thermometer-symbolic";
          "icon-bg-color" = "green";
          "label-color" = "green";
        }
        {
          id = "power";
          "icon-name" = "ld-power-symbolic";
          "icon-show" = true;
          "label-show" = false;
          "icon-bg-color" = "red";
          "left-click" = "~/.config/hypr/scripts/power-menu.sh";
        }
      ];

      "hyprland-workspaces" = {
        "monitor-specific" = true;
        "show-special" = true;
        "display-mode" = "label";
        "label-use-name" = true;
        numbering = "absolute";
        "active-indicator" = "background";
        "active-color" = "yellow";
        "occupied-color" = "fg-muted";
        "empty-color" = "fg-subtle";
        "container-bg-color" = "bg-surface-elevated";
      };

      "window-title" = {
        format = "{{ title }}";
        "label-max-length" = 48;
        "icon-show" = false;
        "label-color" = "fg-muted";
      };

      clock = {
        format = "%m月%d日 %H:%M %a";
        "icon-name" = "tb-calendar-time-symbolic";
        "icon-bg-color" = "yellow";
        "label-color" = "fg";
        "left-click" = "dropdown:calendar";
        "right-click" = "dropdown:weather";
      };

      cpu = {
        "poll-interval-ms" = 3000;
        format = "{{ percent }}%";
        "icon-name" = "ld-cpu-symbolic";
        "icon-bg-color" = "blue";
        "label-color" = "blue";
      };

      ram = {
        "poll-interval-ms" = 5000;
        format = "{{ percent }}%";
        "icon-bg-color" = "red";
        "label-color" = "red";
      };

      media = {
        format = "{{ title }} - {{ artist }}";
        "label-max-length" = 36;
        "left-click" = "dropdown:media";
      };

      volume = {
        format = "{{ percent }}%";
        "left-click" = "dropdown:audio";
        "middle-click" = "wayle audio output-mute";
        "icon-bg-color" = "green";
        "label-color" = "green";
      };

      network = {
        "label-max-length" = 15;
        "left-click" = "dropdown:network";
        "icon-bg-color" = "green";
        "label-color" = "green";
      };

      battery = {
        format = "{{ percent }}%";
        "left-click" = "dropdown:battery";
        "icon-bg-color" = "yellow";
        "label-color" = "yellow";
      };

      bluetooth = {
        "label-max-length" = 15;
        "left-click" = "dropdown:bluetooth";
        "icon-bg-color" = "blue";
        "label-color" = "blue";
      };

      systray = {
        "icon-scale" = 0.9;
        "item-gap" = 0.2;
        "internal-padding" = 0.4;
      };
    };

    osd.enabled = true;
    wallpaper."engine-enabled" = false;
  };
in {
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
    awww  # swww 已 archived 改名 awww（codeberg.org/LGFae/awww）
    wayle
    waybar
    wf-recorder        # 轻量 Wayland 录屏，支持 slurp 区域选择
    wlr-randr
    wl-clipboard
  ];

  wayland.windowManager.hyprland = {
    enable = true;
    # Hyprland 二进制由 NixOS 模块 (programs.hyprland.enable) 提供，
    # 这里设为 null 避免 home.packages 再装一份导致 hyprland-share-picker 等
    # 二进制重复在 PATH 中冲突。HM 仅负责生成配置文件和加载 plugins。
    # 见 HM hyprland 模块 package option 的 extraDescription。
    package = null;
    portalPackage = null;
    # 这里仍然维护 hyprlang 片段（extraConfig 里 source *.conf）。HM 26.05
    # 会按 home.stateVersion 默认切到 Lua，显式固定避免后续 stateVersion 更新后
    # 把这些 source 行写进 hyprland.lua。
    configType = "hyprlang";
    # 系统侧已启用 programs.hyprland.withUWSM = true，UWSM 会自己管理
    # graphical-session.target / wayland-wm@hyprland.target。
    # Hyprland 官方 wiki 明确要求：使用 UWSM 时 HM 这边必须关掉 systemd 集成，
    # 否则与 UWSM 冲突。
    # https://github.com/hyprwm/hyprland-wiki/blob/main/content/Useful%20Utilities/Systemd-start.md
    systemd.enable = false;
    xwayland.enable = true;
    # nixpkgs-unstable 当前的 hyprexpo 0.53.0 会触发本地编译，且不兼容
    # Hyprland 0.54.x；暂时移除，等 nixpkgs 提供兼容缓存包后再启用。
    plugins = [];

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
  xdg.configFile."wayle/config.toml".source = wayleConfig;
  xdg.configFile."waybar/config".source = ./waybar/config;
  xdg.configFile."waybar/style.css".source = ./waybar/style.css;
  xdg.configFile."quickshell" = { source = ./quickshell; recursive = true; };
}
