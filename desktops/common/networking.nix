{lib, ...}: let
  localProfilesFile = /etc/nixos/local/network-manager-profiles.nix;
  localEnvFile = /etc/nixos/local/network-manager.env;
in {
  networking.networkmanager = {
    enable = true;
    wifi.backend = lib.mkDefault "iwd";
    wait-online.enable = false;
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
