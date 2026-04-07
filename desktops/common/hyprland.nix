{self, lib, pkgs, unstable, ...} : {
  services.displayManager.sddm = {
    enable = true;
    wayland.enable = true;
  };

  programs.hyprland = {
    enable = true;
    package = unstable.hyprland;
    portalPackage = unstable.xdg-desktop-portal-hyprland;
    withUWSM = true; 
    xwayland.enable = true;
  };

  # SDDM 会话入口：Hyprland (QuickShell)
  # 通过 env 设置 BAR_BACKEND，UWSM 会继承给 Hyprland → autostart.sh
  services.displayManager.sessionPackages = let
    qs-session = pkgs.writeTextDir
      "share/wayland-sessions/hyprland-quickshell.desktop" ''
        [Desktop Entry]
        Name=Hyprland (QuickShell)
        Comment=Hyprland with QuickShell status bar
        Exec=env BAR_BACKEND=quickshell uwsm start -S hyprland
        Type=Application
      '';
  in [
    (qs-session.overrideAttrs { passthru.providedSessions = [ "hyprland-quickshell" ]; })
  ];

  security.pam.services.hyprlock = {};

  # kwallet-pam：SDDM 登录时用用户密码自动解锁 kwallet，
  # 对 KDE 和 Hyprland 会话都生效（nm-applet 通过 Secret Service D-Bus 存取 WiFi 密码）。
  # plasma6.enable 可能已经设了这个，mkDefault 保证不冲突。
  security.pam.services.sddm.enableKwallet = lib.mkDefault true;
}
