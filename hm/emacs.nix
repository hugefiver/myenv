{pkgs, unstable, ...}: {

  programs.emacs = {
    enable = true;
    package = unstable.emacs-pgtk;
    extraPackages = epkgs: with epkgs; [
      # ── 主题 & UI ──
      doom-themes
      doom-modeline
      nerd-icons
      nerd-icons-dired
      nerd-icons-completion
      dashboard
      solaire-mode
      rainbow-delimiters
      highlight-indent-guides
      page-break-lines         # ^L 显示为水平线

      # ── 视觉增强 ──
      ligature                 # 字体连字 (CaskaydiaCove)
      hl-todo                  # 高亮 TODO/FIXME/HACK
      pulsar                   # 跳转时脉冲当前行
      olivetti                 # org/markdown 居中排版

      # ── 补全框架 (Vertico + Orderless + Marginalia + Consult + Corfu) ──
      vertico
      orderless
      marginalia
      consult
      embark
      embark-consult
      corfu
      cape

      # ── 导航 ──
      which-key
      treemacs
      treemacs-nerd-icons
      avy
      ace-window

      # ── Evil 模式 (安装但默认不启用, C-c e 切换) ──
      evil
      evil-collection
      evil-surround
      evil-commentary

      # ── Git ──
      magit
      diff-hl

      # ── 编辑增强 ──
      undo-tree
      expand-region
      multiple-cursors
      wgrep

      # ── 编程语言 ──
      nix-mode
      markdown-mode
      yaml-mode
      web-mode

      # ── Org ──
      org-superstar

      # ── Dired 增强 ──
      diredfl

      # ── 帮助系统 ──
      helpful

      # ── 终端 ──
      vterm

      # ── Tree-sitter 语法 ──
      treesit-grammars.with-all-grammars
    ];
  };

  # Emacs daemon — 让 emacsclient 秒开
  services.emacs = {
    enable = true;
    startWithUserSession = "graphical";
  };

  xdg.configFile = {
    "emacs/early-init.el".source = ./emacs/early-init.el;
    "emacs/init.el".source = ./emacs/init.el;
    "emacs/cheatsheet.org".source = ./emacs/cheatsheet.org;
    "emacs/banner.txt".source = ./emacs/banner.txt;
  };
}
