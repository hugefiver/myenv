{self, pkgs, ...} : {
  programs.zsh.enable = true;

  users.mutableUsers = true;
  users.users = {
    hugefiver = (import "${self}/users/hugefiver") // {
      initialPassword = "weakpassword";
      shell = pkgs.zsh;
    };
  };
}
