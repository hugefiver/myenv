{
  self,
  lib,
  pkgs,
  unstable,
  ...
}: let
  uwsm = lib.getExe pkgs.uwsm;
  hyprlandSession = pkgs.writeTextDir "share/wayland-sessions/hyprland.desktop" ''
    [Desktop Entry]
    Name=Hyprland
    Comment=Hyprland compositor
    Exec=${uwsm} start -eD Hyprland -F -- ${unstable.hyprland}/bin/start-hyprland
    Type=Application
    DesktopNames=Hyprland
  '';
in {
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

  # QuickShell 入口已知无法启动，暂不作为 SDDM 可选会话暴露。
  services.displayManager.sessionPackages = [
    (hyprlandSession.overrideAttrs {passthru.providedSessions = ["hyprland"];})
  ];

  security.pam.services.hyprlock = {};

  # kwallet-pam：SDDM 登录时用用户密码自动解锁 kwallet，
  # 对 KDE 和 Hyprland 会话都生效（nm-applet 通过 Secret Service D-Bus 存取 WiFi 密码）。
  # plasma6.enable 可能已经设了这个，mkDefault 保证不冲突。
  security.pam.services.sddm.enableKwallet = lib.mkDefault true;
}
