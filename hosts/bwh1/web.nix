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
    package = unstable.caddy;

    virtualHosts = {
      "test" = {
        hostName = "prk.rurilove.moe";

        extraConfig = ''
          handle_path /.well-known/acme-challenge/* {
            root /var/lib/acme/acme-challenge
          }
        '';
      };
    };
  };

  security.acme = {
    acceptTerms = true;
    defaults.email = "no-connect@iruri.moe";
    defaults.webroot = "/var/lib/acme/acme-challenge";

    certs = {
      "test" = {
        domain = "prk.rurilove.moe";
      };
    };
  };
  users.extraGroups.acme.members = ["root"];
}
