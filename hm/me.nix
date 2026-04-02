{self, pkgs, lib, unstable, config, ...} : {

  imports = [
    ./hyprland
    ./kde.nix

    ./wezterm.nix
    ./emacs.nix

    ./hidpi.nix
    ./ime.nix
    ./xdg.nix
    ./gtk.nix
  ];
  
  home.enableNixpkgsReleaseCheck = false;
  home.backupFileExtension = "hm-bak";   # switch 遇到已有文件自动备份为 .hm-bak

  # switch 前处理旧备份：与当前文件相同则删除，不同则保留为 .bak.{N}
  home.activation.removeExistingBackups = lib.hm.dag.entryBefore ["checkLinkTargets"] ''
    find "${config.home.homeDirectory}/.config" -name "*.hm-bak" -type f 2>/dev/null | while IFS= read -r bak; do
      orig="''${bak%.hm-bak}"
      if [ -e "$orig" ] && diff -q "$bak" "$orig" >/dev/null 2>&1; then
        rm -f "$bak"
      else
        n=1
        while [ -e "''${orig}.bak.''${n}" ]; do n=$((n + 1)); done
        mv "$bak" "''${orig}.bak.''${n}"
      fi
    done
  '';
  
  home.username = "hugefiver";
  home.homeDirectory = "/home/hugefiver";
  home.stateVersion = "25.11";

  home.packages = with unstable; [
    fastfetch
    htop
    git
    fzf
    ripgrep
    git-credential-manager
    firefox
    vscode
  ];

  home.sessionVariables = {
    BROWSER = "falkon";
    EDITOR = "emacsclient -c -a emacs";
    TERMINAL = "wezterm";
    VISUAL = "emacsclient -c -a emacs";
  };

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
    shellAliases = {
      hm = "home-manager";
    };
    initContent = ''
      compdef hm=home-manager
    '';
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

  xdg.userDirs = {
    enable = true;
    createDirectories = true;
    desktop = "${config.home.homeDirectory}/Desktop";
    documents = "${config.home.homeDirectory}/Documents";
    download = "${config.home.homeDirectory}/Downloads";
    music = "${config.home.homeDirectory}/Music";
    pictures = "${config.home.homeDirectory}/Pictures";
    publicShare = "${config.home.homeDirectory}/Public";
    templates = "${config.home.homeDirectory}/Templates";
    videos = "${config.home.homeDirectory}/Videos";
  };

  # Force overwrite if file already exists outside home-manager management
  xdg.configFile."user-dirs.dirs".force = true;

  programs.home-manager = {
    enable = true;
  };
}
