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

    globalConfig = ''
      auto_https ignore_loaded_certs
    '';

    extraConfig = ''
      import conf.d/*
    '';
  };
}
