{ pkgs, ... }: {
  # 禁止 KDE 恢复上次 session（避免启动时自动打开设置/文件/终端等窗口）
  xdg.configFile."ksmserverrc".text = ''
    [General]
    loginMode=emptySession
  '';

  home.packages = with pkgs.kdePackages; [
    ark
    dolphin
    falkon
    gwenview
    kate
    okular
    plasma-nm
  ];
}
