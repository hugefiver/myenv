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

  # kwallet-pam：SDDM 登录时用用户密码自动解锁 kwallet，
  # 对 KDE 和 Hyprland 会话都生效（nm-applet 通过 Secret Service D-Bus 存取 WiFi 密码）。
  # plasma6.enable 可能已经设了这个，mkDefault 保证不冲突。
  security.pam.services.sddm.enableKwallet = lib.mkDefault true;
}
