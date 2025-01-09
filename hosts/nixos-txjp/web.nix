{
  pkgs,
  unstable,
  ...
}: {
  # services.nginx = {
  #   enable = true;
  #   package = pkgs.nginxStable.override {
  #     openssl = pkgs.libressl;
  #   };
  # };

  services.caddy = {
    enable = true;
    # package = unstable.caddy;

    extraConfig = ''
      import conf.d/*
    '';
  };
}
