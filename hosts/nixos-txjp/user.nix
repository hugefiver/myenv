{self,...}:{
  users.users = rec {
    root = {
      openssh.authorizedKeys.keys = [] ++ hugefiver.openssh.authorizedKeys.keys;
    };
    hugefiver = import "${self}/users/hugefiver";
  };
}