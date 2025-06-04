{
  isNormalUser = true;
  extraGroups = ["wheel"];
  openssh.authorizedKeys.keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJ2UqF3Qxa7QLckd3y1poViOwjEejmI3J9N2jf4mhUs4 icceey@Studio"
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDTvKGzM54jRRMC6TtMEPRP+zbDOOxxpa+ZFffe50nOJ icceey@PC"
  ];
}
