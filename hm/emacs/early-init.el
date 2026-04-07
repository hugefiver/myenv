;;; early-init.el -*- lexical-binding: t; -*-
;;
;; Runs before init.el — optimize startup and prevent UI flicker.

;; ── Startup Performance ──────────────────────────────────────────
;; Maximize GC threshold during init, restore in init.el
(setq gc-cons-threshold most-positive-fixnum
      gc-cons-percentage 0.6)

;; Detect Nix-managed Emacs (binary lives in /nix/store)
(defvar my/nix-managed-p
  (and invocation-directory
       (string-match-p "/nix/store" invocation-directory))
  "Non-nil if this Emacs was installed via Nix.")

;; Defer package.el — we initialize manually in init.el
(setq package-enable-at-startup nil)

;; Don't process file handlers during init
(defvar default-file-name-handler-alist file-name-handler-alist)
(setq file-name-handler-alist nil)
(add-hook 'emacs-startup-hook
  (lambda ()
    (setq file-name-handler-alist default-file-name-handler-alist)))

;; Suppress native-comp warnings
(setq native-comp-async-report-warnings-errors 'silent)

;; ── Prevent UI Flicker ───────────────────────────────────────────
;; Disable UI chrome BEFORE frame renders
(push '(menu-bar-lines . 0) default-frame-alist)
(push '(tool-bar-lines . 0) default-frame-alist)
(push '(vertical-scroll-bars) default-frame-alist)

;; Prevent flash of light theme — match Gruvbox dark bg
(push '(background-color . "#282828") default-frame-alist)
(push '(foreground-color . "#ebdbb2") default-frame-alist)

;; Frame appearance
(push '(internal-border-width . 8) default-frame-alist)
(push '(undecorated-round . t) default-frame-alist)

;; Avoid expensive frame resizing during startup
(setq frame-inhibit-implied-resize t)

;; Don't compact font caches during GC (improves performance)
(setq inhibit-compacting-font-caches t)

;;; early-init.el ends here
