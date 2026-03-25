{pkgs, ...}: {
  home.packages = [
    pkgs.emacs-pgtk
  ];

  xdg.configFile."emacs/init.el".source = ./emacs/init.el;
}
