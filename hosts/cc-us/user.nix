{self,...}:{
  users.users = rec {
    root = {
      openssh.authorizedKeys.keys = [] ++ hugefiver.openssh.authorizedKeys.keys;
      hashedPassword = "$y$j9T$czocy.6VC6HhOTBn6DvtN0$vNIT4pDcYCjN4qJxgECELF5BZMsMpWlR8meO5M25Os4";
    };
    hugefiver = import "${self}/users/hugefiver";
    yan = import "${self}/users/yan";
  };
}