{
  pkgs,
  unstable,
}: {
  # services.nginx = {
  #   enable = true;
  #   package = pkgs.nginxStable.override {
  #     openssl = pkgs.libressl;
  #   };
  # };

  services.caddy = {
    enable = true;
    package = unstable.caddy;

    virtualHosts = {
      "test" = {
        hostName = "_prk.rurilove.moe";

        extraConfig = ''
          handle_path /.well-known/acme-challenge/* {
            root /var/lib/acme/acme-challenge
            auto_https off
          }
        '';
      };
    };
  };

  security.acme = {
    enable = true;
    email = "no-connect@iruri.moe";

    certs = {
      "test" = {
        domain = "_prk.rurilove.moe";
      };
    };
  };
}
