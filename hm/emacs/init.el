;;; init.el -*- lexical-binding: t; -*-
;;
;; Doom-inspired Emacs config — portable across Nix and non-Nix.
;; Nix 环境: 核心包由 Nix 管理，额外包可从 MELPA 安装
;; 非 Nix 环境: 所有包自动从 MELPA 安装
;; Evil mode installed but NOT enabled by default.
;; Toggle Evil: C-c e  |  Cheatsheet: C-c o
;;
;; ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


;; ┌──────────────────────────────────────────────────────────────┐
;; │                       PERFORMANCE                           │
;; └──────────────────────────────────────────────────────────────┘

;; Restore reasonable GC threshold after startup
(add-hook 'emacs-startup-hook
  (lambda ()
    (setq gc-cons-threshold (* 16 1024 1024)   ; 16 MB
          gc-cons-percentage 0.1)
    (message "Emacs ready in %.2fs with %d GCs."
             (float-time (time-subtract after-init-time before-init-time))
             gcs-done)))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                      BASIC SETTINGS                         │
;; └──────────────────────────────────────────────────────────────┘

(setq inhibit-startup-screen t
      initial-scratch-message ";; Scratch buffer — hack away!\n\n"
      ring-bell-function 'ignore
      make-backup-files nil
      auto-save-default nil
      create-lockfiles nil
      custom-file (expand-file-name "custom.el" user-emacs-directory)
      use-short-answers t
      confirm-kill-emacs #'y-or-n-p
      ;; Scrolling
      scroll-conservatively 101
      scroll-margin 4
      mouse-wheel-scroll-amount '(3 ((shift) . 1))
      mouse-wheel-progressive-speed nil
      ;; Misc
      tab-always-indent 'complete
      require-final-newline t
      sentence-end-double-space nil)

(setq-default indent-tabs-mode nil
              tab-width 2
              fill-column 88
              cursor-type 'bar
              word-wrap t
              truncate-lines t
              line-spacing 0.12)            ; 行间距, 更舒适的阅读体验

;; UTF-8 everywhere
(set-language-environment "UTF-8")
(prefer-coding-system 'utf-8)

;; Built-in modes
(column-number-mode 1)
(global-display-line-numbers-mode 1)
(show-paren-mode 1)
(electric-pair-mode 1)
(savehist-mode 1)
(recentf-mode 1)
(global-auto-revert-mode 1)
(winner-mode 1)
(delete-selection-mode 1)
(global-hl-line-mode 1)
(size-indication-mode 1)
(context-menu-mode 1)
(blink-cursor-mode -1)                    ; Doom 风格: 不闪烁

;; Relative line numbers in prog-mode
(add-hook 'prog-mode-hook (lambda () (setq display-line-numbers 'relative)))

;; Remember cursor position
(save-place-mode 1)

;; Smooth scrolling (Emacs 29+)
(when (fboundp 'pixel-scroll-precision-mode)
  (pixel-scroll-precision-mode 1))

;; Show-paren 增强
(setq show-paren-delay 0.1
      show-paren-highlight-openparen t
      show-paren-when-point-inside-paren t
      show-paren-when-point-in-periphery t)

;; Window dividers (Doom 风格: 细线分隔窗口)
(setq window-divider-default-right-width 1
      window-divider-default-bottom-width 1
      window-divider-default-places 'right-only)
(add-hook 'after-init-hook #'window-divider-mode)

;; Fringe (左右边距指示器)
(when (display-graphic-p)
  (fringe-mode '(8 . 8))
  ;; 用 bitmap 指示截断/续行
  (setq-default indicate-buffer-boundaries 'left
                indicate-empty-lines t))

;; Recentf
(setq recentf-max-saved-items 100
      recentf-max-menu-items 15)

;; Load custom file
(load custom-file 'noerror 'nomessage)


;; ┌──────────────────────────────────────────────────────────────┐
;; │                   PACKAGE MANAGEMENT                        │
;; │  Nix: 核心包已在 load-path, 额外包可 :ensure t 从 MELPA 装  │
;; │  非 Nix: 全部包自动从 MELPA 安装                             │
;; └──────────────────────────────────────────────────────────────┘

(require 'package)
(setq package-archives
      '(("melpa"  . "https://melpa.org/packages/")
        ("gnu"    . "https://elpa.gnu.org/packages/")
        ("nongnu" . "https://elpa.nongnu.org/nongnu/")))
(package-initialize)

;; my/nix-managed-p is defined in early-init.el
(unless (boundp 'my/nix-managed-p)
  (defvar my/nix-managed-p nil))

(if my/nix-managed-p
    ;; Nix 环境: Nix 提供的包不需要 :ensure
    ;; 想装 nixpkgs 没有的包? 加 :ensure t 即可从 MELPA 安装
    (setq use-package-always-ensure nil)
  ;; 非 Nix 环境: 首次启动自动安装所有包
  (unless package-archive-contents
    (package-refresh-contents))
  (unless (package-installed-p 'use-package)
    (package-install 'use-package))
  (setq use-package-always-ensure t))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                          FONTS                              │
;; └──────────────────────────────────────────────────────────────┘

(when (display-graphic-p)
  ;; 主字体
  (set-face-attribute 'default nil
    :font "CaskaydiaCove Nerd Font Mono"
    :height 125
    :weight 'regular)

  ;; 可变宽字体 (org-mode, markdown)
  (set-face-attribute 'variable-pitch nil
    :font "Noto Sans"
    :height 130)

  ;; 等宽字体
  (set-face-attribute 'fixed-pitch nil
    :font "CaskaydiaCove Nerd Font Mono"
    :height 125)

  ;; 中文字体
  (dolist (charset '(han cjk-misc bopomofo))
    (set-fontset-font t charset (font-spec :family "Noto Sans CJK SC")))

  ;; Emoji
  (set-fontset-font t 'symbol "Noto Color Emoji" nil 'prepend))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                       THEME & UI                            │
;; └──────────────────────────────────────────────────────────────┘

;; ── Doom Themes ──
(use-package doom-themes
  :config
  (setq doom-themes-enable-bold t
        doom-themes-enable-italic t)
  (load-theme 'doom-gruvbox t)           ; 暖色深色主题
  (doom-themes-visual-bell-config)
  (doom-themes-org-config)
  (doom-themes-treemacs-config))

;; ── Doom Modeline ──
(use-package doom-modeline
  :hook (after-init . doom-modeline-mode)
  :config
  (setq doom-modeline-height 32
        doom-modeline-bar-width 4
        doom-modeline-icon t
        doom-modeline-major-mode-icon t
        doom-modeline-major-mode-color-icon t
        doom-modeline-buffer-file-name-style 'truncate-upto-project
        doom-modeline-buffer-state-icon t
        doom-modeline-vcs-max-length 20
        doom-modeline-lsp t
        doom-modeline-modal t                ; 显示 Evil 状态
        doom-modeline-modal-icon t           ; Evil 状态图标
        doom-modeline-modal-modern-icon t
        doom-modeline-buffer-encoding nil    ; 隐藏编码 (几乎总是 UTF-8)
        doom-modeline-indent-info nil
        doom-modeline-number-limit 99
        doom-modeline-env-version t))

;; ── Nerd Icons ──
(use-package nerd-icons)

;; ── Nerd Icons + Completion ──
(use-package nerd-icons-completion
  :after marginalia
  :hook (marginalia-mode . nerd-icons-completion-marginalia-setup)
  :config
  (nerd-icons-completion-mode 1))

;; ── Dashboard (启动页) ──
(use-package dashboard
  :config
  (setq dashboard-banner-logo-title "C-c o → Cheatsheet  ·  C-c e → Evil Mode"
        dashboard-startup-banner
          (let ((banner (expand-file-name "banner.txt" user-emacs-directory)))
            (if (file-exists-p banner) banner 'logo))
        dashboard-center-content t
        dashboard-vertically-center-content t
        dashboard-items '((recents   . 8)
                          (projects  . 5)
                          (bookmarks . 5))
        dashboard-set-heading-icons t
        dashboard-set-file-icons t
        dashboard-icon-type 'nerd-icons
        dashboard-projects-backend 'project-el
        dashboard-path-max-length 60
        dashboard-path-style 'truncate-middle
        dashboard-set-navigator t
        dashboard-navigator-buttons
          `(((,(nerd-icons-faicon "nf-fa-github" :height 1.1 :v-adjust 0.0)
              "GitHub" "Browse GitHub"
              (lambda (&rest _) (browse-url "https://github.com")))
             (,(nerd-icons-mdicon "nf-md-cog" :height 1.1 :v-adjust 0.0)
              "Config" "Open init.el"
              (lambda (&rest _) (find-file (expand-file-name "init.el" user-emacs-directory))))
             (,(nerd-icons-mdicon "nf-md-book_open_variant" :height 1.1 :v-adjust 0.0)
              "Cheatsheet" "Open cheatsheet"
              (lambda (&rest _) (find-file (expand-file-name "cheatsheet.org" user-emacs-directory)))))))
  (dashboard-setup-startup-hook))

;; ── Solaire Mode (背景区分) ──
(use-package solaire-mode
  :config
  (solaire-global-mode 1))

;; ── Rainbow Delimiters ──
(use-package rainbow-delimiters
  :hook (prog-mode . rainbow-delimiters-mode))

;; ── Indent Guides ──
(use-package highlight-indent-guides
  :hook (prog-mode . highlight-indent-guides-mode)
  :config
  (setq highlight-indent-guides-method 'bitmap
        highlight-indent-guides-responsive 'top
        highlight-indent-guides-auto-character-face-perc 20))

;; ── Ligatures (字体连字) ──
(use-package ligature
  :config
  (ligature-set-ligatures 'prog-mode
    '(;; CaskaydiaCove / Cascadia Code ligatures
      "|||>" "<|||" "<==>" "<!--" "####" "~~>" "***" "||=" "||>"
      ":::" "::=" "=:=" "===" "==>" "=!=" "=>>" "=<<" "=/=" "!=="
      "!!." ">=>" ">>=" ">>>" ">>-" ">->" "->>" "-->" "---" "-<<"
      "-<-" "<--" "<-<" "<<=" "<<-" "<<<" "<+>" "</>" "###" "#_("
      "..<" "..." "+++" "/==" "///" "_|_" "www" "&&" "^=" "~~" "~@"
      "~=" "~>" "~-" "**" "*>" "*/" "||" "|}" "|]" "|=" "|>" "|-"
      "{|" "[|" "]#" "::" ":=" ":>" ":<" "$>" "==" "=>" "!=" "!!"
      ">:" ">=" ">>" ">-" "-~" "-|" "->" "--" "-<" "<~" "<*" "<|"
      "<:" "<$" "<=" "<>" "<-" "<<" "<+" "</" "#{" "#[" "#:" "#="
      "#!" "#(" "#?" "#_" "%%" ".=" ".-" ".." ".?" "+>" "++" "?:"
      "?=" "?." "??" ";;" "/*" "/=" "/>" "//" "__" "~~" "(*" "*)"
      "\\\\" "://"))
  (global-ligature-mode 1))

;; ── HL-Todo (高亮 TODO/FIXME/HACK) ──
(use-package hl-todo
  :hook ((prog-mode . hl-todo-mode)
         (conf-mode . hl-todo-mode))
  :config
  (setq hl-todo-highlight-punctuation ":"
        hl-todo-keyword-faces
        '(("TODO"       warning bold)
          ("FIXME"      error bold)
          ("HACK"       font-lock-constant-face bold)
          ("REVIEW"     font-lock-keyword-face bold)
          ("NOTE"       success bold)
          ("DEPRECATED" font-lock-doc-face bold)
          ("BUG"        error bold)
          ("XXX"        font-lock-constant-face bold))))

;; ── Pulsar (跳转时脉冲高亮当前行) ──
(use-package pulsar
  :hook (after-init . pulsar-global-mode)
  :config
  (setq pulsar-pulse t
        pulsar-delay 0.055
        pulsar-iterations 10
        pulsar-face 'pulsar-magenta
        pulsar-highlight-face 'pulsar-yellow)
  (dolist (fn '(recenter-top-bottom
                other-window
                ace-window
                windmove-do-window-select
                consult-buffer))
    (add-to-list 'pulsar-pulse-functions fn)))

;; ── Page Break Lines (^L 显示为水平线) ──
(use-package page-break-lines
  :hook (after-init . global-page-break-lines-mode))


;; ┌──────────────────────────────────────────────────────────────┐
;; │               COMPLETION (Vertico Stack)                    │
;; └──────────────────────────────────────────────────────────────┘

;; ── Vertico (竖排候选列表) ──
(use-package vertico
  :hook (after-init . vertico-mode)
  :config
  (setq vertico-count 12
        vertico-cycle t
        vertico-resize nil))

;; ── Orderless (模糊匹配) ──
(use-package orderless
  :config
  (setq completion-styles '(orderless basic)
        completion-category-defaults nil
        completion-category-overrides '((file (styles partial-completion)))))

;; ── Marginalia (候选注释) ──
(use-package marginalia
  :hook (after-init . marginalia-mode))

;; ── Consult (增强搜索/跳转) ──
(use-package consult
  :bind (("C-s"     . consult-line)          ; 替代 isearch
         ("C-r"     . consult-line)          ; 反向也用 consult
         ("C-x b"   . consult-buffer)        ; 增强 buffer 切换
         ("C-x r b" . consult-bookmark)
         ("M-g g"   . consult-goto-line)
         ("M-g M-g" . consult-goto-line)
         ("M-g o"   . consult-outline)       ; 跳到大纲
         ("M-g i"   . consult-imenu)         ; 跳到符号
         ("M-s r"   . consult-ripgrep)       ; 全项目 ripgrep
         ("M-s f"   . consult-find)          ; 全项目 find
         ("M-s l"   . consult-line)
         ("M-y"     . consult-yank-pop))     ; 剪贴板历史
  :config
  (setq consult-narrow-key "<"
        consult-preview-key "M-."))

;; ── Embark (上下文操作) ──
(use-package embark
  :bind (("C-."   . embark-act)
         ("C-;"   . embark-dwim)
         ("C-h B" . embark-bindings)))

(use-package embark-consult
  :hook (embark-collect-mode . consult-preview-at-point-mode))

;; ── Corfu (行内补全) ──
(use-package corfu
  :hook (after-init . global-corfu-mode)
  :config
  (setq corfu-auto t
        corfu-auto-delay 0.2
        corfu-auto-prefix 2
        corfu-cycle t
        corfu-preselect 'prompt
        corfu-popupinfo-delay '(0.5 . 0.2))
  (corfu-popupinfo-mode 1))

;; ── Cape (补全后端) ──
(use-package cape
  :init
  (add-hook 'completion-at-point-functions #'cape-dabbrev)
  (add-hook 'completion-at-point-functions #'cape-file)
  (add-hook 'completion-at-point-functions #'cape-keyword))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                       NAVIGATION                            │
;; └──────────────────────────────────────────────────────────────┘

;; ── Which Key (按键提示) ──
(use-package which-key
  :hook (after-init . which-key-mode)
  :config
  (setq which-key-idle-delay 0.5
        which-key-sort-order 'which-key-key-order-alpha
        which-key-add-column-padding 2))

;; ── Treemacs (文件树) ──
(use-package treemacs
  :bind (("<f8>" . treemacs)
         ("C-x t t" . treemacs)
         ("C-x t 1" . treemacs-select-window))
  :config
  (setq treemacs-width 30
        treemacs-is-never-other-window t
        treemacs-show-hidden-files t
        treemacs-follow-after-init t))

(use-package treemacs-nerd-icons
  :after treemacs
  :config
  (treemacs-load-theme "nerd-icons"))

;; ── Avy (快速跳转) ──
(use-package avy
  :bind (("M-j"   . avy-goto-char-timer)
         ("M-g l" . avy-goto-line)
         ("M-g w" . avy-goto-word-1)))

;; ── Ace Window (窗口跳转) ──
(use-package ace-window
  :bind ("M-o" . ace-window)
  :config
  (setq aw-keys '(?a ?s ?d ?f ?g ?h ?j ?k ?l)
        aw-scope 'frame))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                  EVIL MODE (默认关闭)                        │
;; │           C-c e 切换  |  主要用于浏览文件                        │
;; └──────────────────────────────────────────────────────────────┘

(use-package evil
  :init
  (setq evil-want-integration t
        evil-want-keybinding nil          ; evil-collection 需要
        evil-want-C-u-scroll t
        evil-want-Y-yank-to-eol t
        evil-undo-system 'undo-tree
        evil-respect-visual-line-mode t
        evil-split-window-below t
        evil-vsplit-window-right t)
  :config
  ;; Evil 默认不启用 — C-c e 手动切换
  ;; 启用后 SPC 作为 leader key
  (evil-set-leader 'normal (kbd "SPC"))

  ;; Leader 键位 (仅 evil-mode 启用后生效)
  (evil-define-key 'normal 'global
    (kbd "<leader>ff") #'consult-find
    (kbd "<leader>fr") #'consult-recent-file
    (kbd "<leader>fb") #'consult-buffer
    (kbd "<leader>fg") #'consult-ripgrep
    (kbd "<leader>fl") #'consult-line
    (kbd "<leader>bb") #'consult-buffer
    (kbd "<leader>bd") #'kill-current-buffer
    (kbd "<leader>ww") #'ace-window
    (kbd "<leader>wd") #'delete-window
    (kbd "<leader>ws") #'split-window-below
    (kbd "<leader>wv") #'split-window-right
    (kbd "<leader>gg") #'magit-status
    (kbd "<leader>tt") #'treemacs
    (kbd "<leader>tv") #'vterm
    (kbd "<leader>pp") #'project-switch-project
    (kbd "<leader>pf") #'project-find-file
    (kbd "<leader>hh") (lambda () (interactive)
                         (find-file (expand-file-name "cheatsheet.org" user-emacs-directory)))
    (kbd "<leader>u")  #'universal-argument))

(use-package evil-collection
  :after evil
  :config
  (evil-collection-init))

(use-package evil-surround
  :after evil
  :hook (evil-local-mode . evil-surround-mode))

(use-package evil-commentary
  :after evil
  :hook (evil-local-mode . evil-commentary-mode))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                           GIT                               │
;; └──────────────────────────────────────────────────────────────┘

(use-package magit
  :bind ("C-x g" . magit-status)
  :config
  (setq magit-display-buffer-function
        #'magit-display-buffer-same-window-except-diff-v1))

(use-package diff-hl
  :hook ((after-init         . global-diff-hl-mode)
         (magit-post-refresh . diff-hl-magit-post-refresh)
         (dired-mode         . diff-hl-dired-mode))
  :config
  (diff-hl-flydiff-mode 1))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                        EDITING                              │
;; └──────────────────────────────────────────────────────────────┘

(use-package undo-tree
  :hook (after-init . global-undo-tree-mode)
  :config
  (setq undo-tree-history-directory-alist
        `(("." . ,(expand-file-name "undo-tree/" user-emacs-directory)))
        undo-tree-auto-save-history t
        undo-tree-visualizer-timestamps t))

(use-package expand-region
  :bind ("C-=" . er/expand-region))

(use-package multiple-cursors
  :bind (("C-S-c C-S-c" . mc/edit-lines)
         ("C->"         . mc/mark-next-like-this)
         ("C-<"         . mc/mark-previous-like-this)
         ("C-c C-<"     . mc/mark-all-like-this)))

(use-package wgrep
  :config
  (setq wgrep-auto-save-buffer t))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                      PROGRAMMING                            │
;; └──────────────────────────────────────────────────────────────┘

;; ── Eglot (LSP, Emacs 29+ 内置) ──
(use-package eglot
  :ensure nil
  :hook ((nix-mode         . eglot-ensure)
         (python-mode      . eglot-ensure)
         (python-ts-mode   . eglot-ensure)
         (js-mode          . eglot-ensure)
         (js-ts-mode       . eglot-ensure)
         (typescript-ts-mode . eglot-ensure)
         (tsx-ts-mode      . eglot-ensure)
         (rust-mode        . eglot-ensure)
         (go-mode          . eglot-ensure)
         (go-ts-mode       . eglot-ensure)
         (c-mode           . eglot-ensure)
         (c-ts-mode        . eglot-ensure)
         (c++-mode         . eglot-ensure)
         (c++-ts-mode      . eglot-ensure))
  :config
  (setq eglot-autoshutdown t
        eglot-events-buffer-size 0
        eglot-send-changes-idle-time 0.5))

;; ── Tree-sitter (Emacs 29+ 内置) ──
(when (treesit-available-p)
  (setq treesit-font-lock-level 4)
  (setq major-mode-remap-alist
        '((python-mode     . python-ts-mode)
          (javascript-mode . js-ts-mode)
          (typescript-mode . typescript-ts-mode)
          (json-mode       . json-ts-mode)
          (yaml-mode       . yaml-ts-mode)
          (css-mode        . css-ts-mode)
          (bash-mode       . bash-ts-mode)
          (sh-mode         . bash-ts-mode)
          (c-mode          . c-ts-mode)
          (c++-mode        . c++-ts-mode)
          (cmake-mode      . cmake-ts-mode)
          (toml-mode       . toml-ts-mode)
          (rust-mode       . rust-ts-mode)
          (go-mode         . go-ts-mode))))

;; ── Language Modes ──
(use-package nix-mode
  :mode "\\.nix\\'")

(use-package markdown-mode
  :mode ("\\.md\\'" "\\.markdown\\'")
  :config
  (setq markdown-fontify-code-blocks-natively t))

(use-package yaml-mode
  :mode ("\\.yml\\'" "\\.yaml\\'"))

(use-package web-mode
  :mode ("\\.html\\'" "\\.vue\\'" "\\.svelte\\'" "\\.jsx\\'" "\\.tsx\\'")
  :config
  (setq web-mode-markup-indent-offset 2
        web-mode-css-indent-offset 2
        web-mode-code-indent-offset 2))

;; ── Flymake (语法检查, 内置) ──
(use-package flymake
  :ensure nil
  :hook (prog-mode . flymake-mode)
  :bind (:map flymake-mode-map
         ("M-n" . flymake-goto-next-error)
         ("M-p" . flymake-goto-prev-error)))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                        ORG MODE                             │
;; └──────────────────────────────────────────────────────────────┘

(use-package org
  :config
  (setq org-startup-indented t
        org-hide-leading-stars t
        org-ellipsis " ▾"
        org-pretty-entities t
        org-return-follows-link t
        org-src-fontify-natively t
        org-src-tab-acts-natively t
        org-startup-folded 'content
        org-confirm-babel-evaluate nil
        org-log-done 'time))

(use-package org-superstar
  :hook (org-mode . org-superstar-mode)
  :config
  (setq org-superstar-headline-bullets-list '("◉" "○" "●" "◆" "▶")
        org-superstar-item-bullet-alist '((?- . ?•) (?+ . ?➤))))

;; ── Olivetti (居中排版, Doom 风格沉浸式写作) ──
(use-package olivetti
  :hook ((org-mode      . olivetti-mode)
         (markdown-mode . olivetti-mode))
  :config
  (setq olivetti-body-width 88
        olivetti-minimum-body-width 72
        olivetti-style 'fancy))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                          DIRED                              │
;; └──────────────────────────────────────────────────────────────┘

(use-package dired
  :ensure nil
  :config
  (setq dired-listing-switches "-agho --group-directories-first"
        dired-dwim-target t
        dired-recursive-copies 'always
        dired-recursive-deletes 'always
        delete-by-moving-to-trash t
        dired-kill-when-opening-new-dired-buffer t))

(use-package diredfl
  :hook (dired-mode . diredfl-mode))

(use-package nerd-icons-dired
  :hook (dired-mode . nerd-icons-dired-mode))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                       HELP SYSTEM                           │
;; └──────────────────────────────────────────────────────────────┘

(use-package helpful
  :bind (("C-h f" . helpful-callable)
         ("C-h v" . helpful-variable)
         ("C-h k" . helpful-key)
         ("C-h x" . helpful-command)
         ("C-h o" . helpful-symbol)))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                        TERMINAL                             │
;; └──────────────────────────────────────────────────────────────┘

(use-package vterm
  :bind ("C-x t v" . vterm)
  :config
  (setq vterm-max-scrollback 10000
        vterm-timer-delay 0.01))


;; ┌──────────────────────────────────────────────────────────────┐
;; │                    GLOBAL KEYBINDINGS                       │
;; └──────────────────────────────────────────────────────────────┘

;; F5  = 刷新 buffer
;; F8  = Treemacs (文件树)
;; C-c e = 切换 Evil 模式
;; C-c o = 打开 Cheatsheet

(global-set-key (kbd "C-x k") #'kill-current-buffer)
(global-set-key (kbd "<f5>")  #'revert-buffer-quick)
(global-set-key (kbd "C-c e") #'evil-mode)
(global-set-key (kbd "<escape>") #'keyboard-escape-quit)

;; 打开 cheatsheet
(global-set-key (kbd "C-c o")
  (lambda () (interactive)
    (find-file (expand-file-name "cheatsheet.org" user-emacs-directory))))

;; 快速访问 init.el
(global-set-key (kbd "C-c i")
  (lambda () (interactive)
    (find-file (expand-file-name "init.el" user-emacs-directory))))


;;; init.el ends here
