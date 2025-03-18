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
      # "test" = {
      #   domain = "prk.rurilove.moe";
      #   dnsProvider = "cloudflare";
      #   credentialFiles = {
      #     "CF_DNS_API_TOKEN_FILE" = "/root/.pri/cf_api_key";
      #   };
      #   webroot = null;
      # };
    };
  };
  users.extraGroups.acme.members = ["root"];
}
