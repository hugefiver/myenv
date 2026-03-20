{self, pkgs, unstable, config, ...} : {

  imports = [
    ./ime.nix
  ];
  
  home.username = "hugefiver";
  home.homeDirectory = "/home/hugefiver";
  home.stateVersion = "25.11";

  home.packages = with unstable; [
    fastfetch
    htop
    git
    fzf
    emacs
    ripgrep
    git-credential-manager
  ];

  programs.git = {
    enable = true;
    package = unstable.git;

    settings.user = {
      name = "Hugefiver";
      email = "i@iruri.moe";
    };

  };

  programs.zsh = {
    enable = true;
    oh-my-zsh = {
      enable = true;
      package = unstable.oh-my-zsh;
      plugins = [
        "git"
        "docker"
      ];
      # theme = "frontcube" # "jnrowe" "kphoen"
      theme = "jispwoso";
    };
  };

  programs.home-manager.enable = true;
}
