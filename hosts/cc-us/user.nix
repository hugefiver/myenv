{self,...}:{
  users.users = rec {
    root = {
      openssh.authorizedKeys.keys = [] ++ hugefiver.openssh.authorizedKeys.keys;
      hashedPassword = "$6$2qy5fSYu5wIMGFGL$ZKFFhO7T6SoECqr1BM4QhkAcLB.eCDI3Ejczq6h30iybVIeLXSmV5y/CI.9cWv8KWw/7L36aHGZmCZpgTBH/s1";
    };
    hugefiver = import "${self}/users/hugefiver";
    yan = import "${self}/users/yan";
  };

  users.extraGroups.docker.members = ["root" "hugefiver" "yan"];
}