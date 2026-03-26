{self, lib, pkgs, unstable, ...} : {
  services.displayManager.sddm = {
    enable = true;
    wayland.enable = true;
  };

  programs.hyprland = {
    enable = true;
    withUWSM = true; 
    xwayland.enable = true;
  };

  security.pam.services.hyprlock = {};

  # gnome-keyring 提供 Secret Service D-Bus API，nm-applet 用它存取 WiFi 密码。
  # PAM 集成使得 SDDM 登录时自动解锁 keyring（无需二次输密码）。
  services.gnome.gnome-keyring.enable = true;

  # Hyprland 会话需要显式启动 gnome-keyring-daemon；
  # UWSM 会执行 dbus-update-activation-environment，但 keyring 需要 PAM 先初始化。
  security.pam.services.sddm.enableGnomeKeyring = true;
}
