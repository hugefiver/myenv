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

  environment.systemPackages = with unstable; [
    rofi
    waybar
    dunst
    swww
    grim
    slurp
    wl-clipboard
    cliphist
    brightnessctl
    playerctl
    networkmanagerapplet
    pavucontrol
    kitty
    nautilus
  ];
}
