{lib, ...}: {
  disko.devices = {
    disk.vda = {
      device = lib.mkDefault "/dev/vda";
      type = "disk";
      content = {
        type = "gpt";
        partitions = {
          boot = { size = "1M"; type = "EF02"; };
          ESP = { size = "200M"; type = "EF00"; priority = 2; content = { type = "filesystem"; format = "vfat"; mountpoint = "/boot"; mountOptions = [ "umask=0077" ]; }; };
          root = { name = "root"; size = "100%"; content = { type = "filesystem"; format = "btrfs"; mountpoint = "/"; }; };
        };
      };
    };
  };
}