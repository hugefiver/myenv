{lib, ...}: let
  localProfilesFile = /etc/nixos/local/network-manager-profiles.nix;
  localEnvFile = /etc/nixos/local/network-manager.env;
in {
  # networking.networkmanager.wait-online 在部分 nixpkgs 版本中不存在，
  # 直接通过 systemd 禁用该服务以避免启动等待超时。
  systemd.services.NetworkManager-wait-online.enable = false;

  networking.networkmanager = {
    enable = true;
    wifi.backend = lib.mkDefault "iwd";
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

  security.polkit.enable = true;

  users.users.hugefiver.extraGroups = lib.mkAfter [
    "networkmanager"
  ];
}
