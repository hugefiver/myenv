{lib, ...}: let
  localProfilesFile = /etc/nixos/local/network-manager-profiles.nix;
  localEnvFile = /etc/nixos/local/network-manager.env;
in {
  # 直接通过 systemd 禁用该服务以避免启动等待超时。
  systemd.services.NetworkManager-wait-online.enable = false;
  systemd.services.NetworkManager-wait-online-initrd.enable = false;

  # systemd-networkd-wait-online 也一并禁掉，防止 wlo1 拖慢 network-online.target。
  systemd.network.wait-online.enable = false;

  networking.networkmanager = {
    enable = true;
    wifi.backend = lib.mkDefault "iwd";
    wifi.powersave = false;
    ensureProfiles = {
      environmentFiles = lib.optional (builtins.pathExists localEnvFile) (toString localEnvFile);
      profiles =
        if builtins.pathExists localProfilesFile
        then import localProfilesFile
        else {};
    };
  };

  networking.wireless.iwd = {
    enable = true;
    settings = {
      General = {
        # 使用 NM+iwd 后端时，IP 配置由 NetworkManager 负责；
        # 若这里同时设为 true，iwd 也会抢着做 DHCP，导致 wlo1 启动卡住。
        EnableNetworkConfiguration = false;
      };
      Settings = {
        AutoConnect = true;
      };
    };
  };

  # iwd 启动不应无限阻塞引导，但也不能太激进（WiFi 固件加载可能需要时间）。
  # 默认 systemd timeout 是 90s，30s 足够大多数固件加载。
  systemd.services.iwd.serviceConfig.TimeoutStartSec = lib.mkDefault "30s";

  # ── iwd ↔ NetworkManager 启动顺序 ──────────────────────────
  # 默认 iwd 是 D-Bus activated（按需启动），但 NM 不一定会等 iwd 就绪。
  # 显式将 iwd 加入 multi-user.target 并让 NM 依赖它，消除竞态。
  systemd.services.iwd.wantedBy = [ "multi-user.target" ];
  systemd.services.NetworkManager.after = [ "iwd.service" ];

  security.polkit.enable = true;

  users.users.hugefiver.extraGroups = lib.mkAfter [
    "networkmanager"
  ];
}
