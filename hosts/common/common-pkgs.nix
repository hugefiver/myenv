{pkgs, ...}: {
  environment.systemPackages = with pkgs; [
    vim
    git
    htop
    tmux
    curl
  ];

  # enable bash history search
  programs.bash.interactiveShellInit = ''
    bind '"\e[A": history-search-backward'
    bind '"\e[B": history-search-forward'
  '';
}
