{
  channel ? "nixos-24.05",
  system ? "x86_64-linux",
  mirror ? null,
  useUnstable ? false
}: let
  nixpkgs-unstable.url =
    if mirror != null
    then
      (
        if mirror == "tsinghua" || mirror == "tuna"
        then "git+https://mirrors.tuna.tsinghua.edu.cn/git/nixpkgs.git?ref=nixpkgs-unstable"
        else mirror.unstable
      )
    else "github:nixos/nixpkgs/nixpkgs-unstable";

  nixpkgs-stable.url =
    if mirror != null
    then
      (
        if mirror == "tsinghua" || mirror == "tuna"
        then "git+https://mirrors.tuna.tsinghua.edu.cn/git/nixpkgs.git?ref=${channel}"
        else mirror.stable
      )
    else "github:nixos/nixpkgs/${channel}";
in rec {
  stable = import nixpkgs-stable { inherit system; };
  unstable = import nixpkgs-unstable { inherit system; };
  nixpkgs = if useUnstable then unstable else stable;
  pkgs = nixpkgs;
}
