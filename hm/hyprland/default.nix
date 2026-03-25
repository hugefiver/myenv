{ pkgs, ... }: {
	home.packages = with pkgs; [
		brightnessctl
		cliphist
		dunst
		grim
		libnotify
		networkmanagerapplet
		pavucontrol
		playerctl
		rofi
		slurp
		swww
		waybar
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
	xdg.configFile."hypr/base.conf".source = ./config.txt;
	xdg.configFile."hypr/keybinds.conf".source = ./keybinds.conf;
		xdg.configFile."hypr/monitors.conf".source = ./monitors.conf;
		xdg.configFile."hypr/windowrules.conf".source = ./windowrules.conf;
		xdg.configFile."dunst/dunstrc".source = ./dunst/dunstrc;
	xdg.configFile."rofi/config.rasi".source = ./rofi/config.rasi;
	xdg.configFile."waybar/config".source = ./waybar/config;
	xdg.configFile."waybar/style.css".source = ./waybar/style.css;
		xdg.configFile."wlogout/layout".source = ./wlogout/layout;
		xdg.configFile."wlogout/style.css".source = ./wlogout/style.css;
}
