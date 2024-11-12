{pkgs, ...}: {
  programs.nix-ld = {
    package = pkgs.nix-ld;
    enable = true;
  };
}