{ ... }: {
  # mihomo-boot ⇆ clash-verge GUI 接管（KDE/Hyprland 通用，systemd 同步保证无竞争）
  systemd.user.services.mihomo-boot-handoff = {
    Unit = {
      Description = "Hand off boot-time mihomo to clash-verge GUI";
      After = [ "graphical-session-pre.target" ];
      PartOf = [ "graphical-session.target" ];
    };
    Service = {
      Type = "oneshot";
      RemainAfterExit = true;
      ExecStart = "/run/current-system/sw/bin/systemctl stop mihomo-boot.service";
      ExecStop  = "/run/current-system/sw/bin/systemctl start mihomo-boot.service";
    };
    Install.WantedBy = [ "graphical-session.target" ];
  };

  systemd.user.services.clash-verge = {
    Unit = {
      Description = "Clash Verge Rev (TUN proxy GUI)";
      After = [ "mihomo-boot-handoff.service" "graphical-session.target" ];
      Requires = [ "mihomo-boot-handoff.service" ];
      PartOf = [ "graphical-session.target" ];
    };
    Service = {
      Type = "simple";
      # 等 DisplayLink 出图稳定，避免 webview 首帧按 0×0 算布局
      ExecStartPre = "/run/current-system/sw/bin/sleep 5";
      ExecStart = "/run/current-system/sw/bin/clash-verge";
      Restart = "on-failure";
      RestartSec = "5s";
    };
    Install.WantedBy = [ "graphical-session.target" ];
  };
}
