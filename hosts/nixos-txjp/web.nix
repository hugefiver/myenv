{pkgs, ...}: {
  services.nginx = {
    enable = true;
    package = pkgs.nginxStable.override {
      openssl = pkgs.libressl;
    };

    virtualHosts = {
      "test" = {
        serverName = "_prk.rurilove.moe";
        # useACMEHost = true;
        enableACME = true;
        quic = true;
        reuseport = true;
      };
    };
  };

  # security.acme = {
  #   enable = true;
  #   email = "no-connect@iruri.moe";

  #   certs = {
  #     "test" = {
  #       domain = "_prk.rurilove.moe";
  #     };
  #   };
  # };
}
