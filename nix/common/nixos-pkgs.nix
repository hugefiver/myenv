{pkgs, ...}: {
  environment.systemPackages = with pkgs; [
    legacyPackages.${system}.nixos-install-tools
  ];
}