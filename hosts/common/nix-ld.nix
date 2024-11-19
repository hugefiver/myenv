{pkgs, ...}: {
  programs.nix-ld = {
    enable = true;
    package = pkgs.nix-ld;
    libraries = with pkgs; [ libressl glibc ];
  };
}