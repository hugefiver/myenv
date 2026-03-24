{self, pkgs, ...} : {
  programs.zsh.enable = true;

  users.mutableUsers = true;
  users.users = {
    root = {
      home = "/root";
      initialHashedPassword = "$y$j9T$734Y1uzwCT/ipwXODTe9b/$dmDQvh9mCOusrtDwvjXPRxr4vs06.TEb4fm5oI1jY9/";
      # hashedPasswordFile = "~/.secrets/root-password";
    };
    hugefiver = (import "${self}/users/hugefiver") // {
      initialPassword = "weakpassword";
      # hashedPasswordFile = "~/.secrets/hugefiver-password";
      shell = pkgs.zsh;
    };
  };
}
