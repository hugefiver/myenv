{ pkgs, ... }: {
  xdg.portal = {
    enable = true;
    xdgOpenUsePortal = true;
    extraPortals = [
      pkgs.xdg-desktop-portal-hyprland
      pkgs.kdePackages.xdg-desktop-portal-kde
      pkgs.xdg-desktop-portal-gtk
    ];
    config = {
      common.default = [ "kde" "gtk" ];
      hyprland.default = [ "hyprland" "kde" "gtk" ];
    };
  };

  xdg.mime.enable = true;
  xdg.mimeApps = {
    enable = true;
    defaultApplications = {
      "text/plain" = [ "org.kde.kate.desktop" ];
      "text/markdown" = [ "org.kde.kate.desktop" ];
      "application/json" = [ "org.kde.kate.desktop" ];
      "application/xml" = [ "org.kde.kate.desktop" ];
      "text/xml" = [ "org.kde.kate.desktop" ];

      "image/bmp" = [ "org.kde.gwenview.desktop" ];
      "image/gif" = [ "org.kde.gwenview.desktop" ];
      "image/jpeg" = [ "org.kde.gwenview.desktop" ];
      "image/jpg" = [ "org.kde.gwenview.desktop" ];
      "image/pjpeg" = [ "org.kde.gwenview.desktop" ];
      "image/png" = [ "org.kde.gwenview.desktop" ];
      "image/tiff" = [ "org.kde.gwenview.desktop" ];
      "image/webp" = [ "org.kde.gwenview.desktop" ];
      "image/x-bmp" = [ "org.kde.gwenview.desktop" ];
      "image/x-gray" = [ "org.kde.gwenview.desktop" ];
      "image/x-icb" = [ "org.kde.gwenview.desktop" ];
      "image/x-ico" = [ "org.kde.gwenview.desktop" ];
      "image/x-png" = [ "org.kde.gwenview.desktop" ];
      "image/x-portable-anymap" = [ "org.kde.gwenview.desktop" ];
      "image/x-portable-bitmap" = [ "org.kde.gwenview.desktop" ];
      "image/x-portable-graymap" = [ "org.kde.gwenview.desktop" ];
      "image/x-portable-pixmap" = [ "org.kde.gwenview.desktop" ];
      "image/x-xbitmap" = [ "org.kde.gwenview.desktop" ];
      "image/x-xpixmap" = [ "org.kde.gwenview.desktop" ];
      "image/x-pcx" = [ "org.kde.gwenview.desktop" ];
      "image/svg+xml" = [ "org.kde.gwenview.desktop" ];
      "image/svg+xml-compressed" = [ "org.kde.gwenview.desktop" ];
      "image/vnd.wap.wbmp" = [ "org.kde.gwenview.desktop" ];
      "image/x-icns" = [ "org.kde.gwenview.desktop" ];

      "application/pdf" = [ "org.kde.okular.desktop" ];

      "text/html" = [ "firefox.desktop" ];
      "application/xhtml+xml" = [ "firefox.desktop" ];
      "application/vnd.mozilla.xul+xml" = [ "firefox.desktop" ];
      "x-scheme-handler/http" = [ "firefox.desktop" ];
      "x-scheme-handler/https" = [ "firefox.desktop" ];

      "inode/directory" = [ "org.kde.dolphin.desktop" ];
      "application/x-7z-compressed" = [ "org.kde.ark.desktop" ];
      "application/x-7z-compressed-tar" = [ "org.kde.ark.desktop" ];
      "application/x-bzip" = [ "org.kde.ark.desktop" ];
      "application/x-bzip-compressed-tar" = [ "org.kde.ark.desktop" ];
      "application/x-compress" = [ "org.kde.ark.desktop" ];
      "application/x-compressed-tar" = [ "org.kde.ark.desktop" ];
      "application/x-cpio" = [ "org.kde.ark.desktop" ];
      "application/x-gzip" = [ "org.kde.ark.desktop" ];
      "application/x-lha" = [ "org.kde.ark.desktop" ];
      "application/x-lzip" = [ "org.kde.ark.desktop" ];
      "application/x-lzip-compressed-tar" = [ "org.kde.ark.desktop" ];
      "application/x-lzma" = [ "org.kde.ark.desktop" ];
      "application/x-lzma-compressed-tar" = [ "org.kde.ark.desktop" ];
      "application/x-tar" = [ "org.kde.ark.desktop" ];
      "application/x-tarz" = [ "org.kde.ark.desktop" ];
      "application/x-xar" = [ "org.kde.ark.desktop" ];
      "application/x-xz" = [ "org.kde.ark.desktop" ];
      "application/x-xz-compressed-tar" = [ "org.kde.ark.desktop" ];
      "application/zip" = [ "org.kde.ark.desktop" ];
      "application/gzip" = [ "org.kde.ark.desktop" ];
      "application/bzip2" = [ "org.kde.ark.desktop" ];
      "application/vnd.rar" = [ "org.kde.ark.desktop" ];
      "application/zstd" = [ "org.kde.ark.desktop" ];
      "application/x-zstd-compressed-tar" = [ "org.kde.ark.desktop" ];

      # "x-scheme-handler/tg" = [ "org.telegram.desktop.desktop" ];
      # "message/rfc822" = [ "thunderbird.desktop" ];
      # "x-scheme-handler/mailto" = [ "thunderbird.desktop" ];
      # "text/calendar" = [ "thunderbird.desktop" ];
      # "text/x-vcard" = [ "thunderbird.desktop" ];
    };
  };
}