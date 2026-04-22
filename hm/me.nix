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

  # switch 前：HM 管理的路径若已有非符号链接文件，对比后备份/删除，避免冲突
  home.activation.resolveFileConflicts = lib.hm.dag.entryBefore ["checkLinkTargets"] ''
    hm_gendir="''${XDG_STATE_HOME:-$HOME/.local/state}/nix/profiles/home-manager"
    new_gen=$(readlink -f "$hm_gendir") 2>/dev/null || true

    if [ -d "$new_gen/home-files" ]; then
      find "$new_gen/home-files" -type f -o -type l | while IFS= read -r store_file; do
        rel="''${store_file#$new_gen/home-files/}"
        target="$HOME/$rel"

        # 只处理普通文件冲突（符号链接是 HM 自己管理的，不冲突）
        [ -e "$target" ] && [ ! -L "$target" ] || continue

        if diff -q "$target" "$store_file" >/dev/null 2>&1; then
          rm -f "$target"
        else
          n=1
          while [ -e "''${target}.bak.''${n}" ]; do n=$((n + 1)); done
          mv "$target" "''${target}.bak.''${n}"
        fi
      done
    fi
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
    BROWSER = "firefox";
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
