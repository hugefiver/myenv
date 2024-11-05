{pkgs, ...}: {
  options.common-pkgs.enable = true;

  environment.systemPackages = with pkgs; [
    git
    vim
    tmux
    curl
    htop
  ];

  environment.variables.EDITOR = "vim";
}
