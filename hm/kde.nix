{ pkgs, ... }: {
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
