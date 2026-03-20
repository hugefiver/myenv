{pkgs, ...}: {
  environment.systemPackages = with pkgs; [
    vim
    git
    htop
    tmux
    curl
    # curlWithGnuTls
  ];

  # enable bash history search
  programs.bash.interactiveShellInit = ''
    bind '"\e[A": history-search-backward'
    bind '"\e[B": history-search-forward'
  '';

  programs.nix-ld = {
    enable = true;
    package = pkgs.nix-ld;
    libraries = with pkgs; [ libressl glibc ];
  };
}
