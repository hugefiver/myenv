(setq inhibit-startup-screen t
      initial-scratch-message nil
      ring-bell-function 'ignore
      make-backup-files nil
      auto-save-default nil
      create-lockfiles nil
      custom-file (expand-file-name "custom.el" user-emacs-directory))

(menu-bar-mode -1)
(tool-bar-mode -1)
(scroll-bar-mode -1)
(column-number-mode 1)
(global-display-line-numbers-mode 1)
(show-paren-mode 1)
(electric-pair-mode 1)
(savehist-mode 1)
(recentf-mode 1)
(global-auto-revert-mode 1)
(winner-mode 1)
(fido-vertical-mode 1)

(when (fboundp 'pixel-scroll-precision-mode)
  (pixel-scroll-precision-mode 1))

(setq-default indent-tabs-mode nil
              tab-width 2
              fill-column 88)

(when (display-graphic-p)
  (set-face-attribute 'default nil :font "CaskaydiaCove Nerd Font Mono-12"))

(load custom-file 'noerror 'nomessage)
