{pkgs, ...}: {
  environment.systemPackages = with pkgs; [
    vim
    git
    htop
    tmux
    curl
  ];
}
