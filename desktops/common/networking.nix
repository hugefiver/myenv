{lib, ...}: let
  localProfilesFile = /etc/nixos/local/network-manager-profiles.nix;
  localEnvFile = /etc/nixos/local/network-manager.env;
in {
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
        EnableNetworkConfiguration = true;
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
