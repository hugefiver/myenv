{pkgs, unstable, ...}: {
  services.nginx = {
    enable = true;
    package = pkgs.nginxStable.override {
      openssl = pkgs.libressl;
    };
    
  }
}