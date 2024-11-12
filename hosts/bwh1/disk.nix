{lib, ...}: {
  disko.devices = {
    disk.sda = {
      device = lib.mkDefault "/dev/sda";
      type = "disk";
      content = {
        type = "gpt";
        partitions = {
           boot = { size = "1M"; type = "EF02"; };
           root = { name = "root"; size = "100%"; content = { type = "filesystem"; format = "btrfs"; mountpoint = "/"; }; };
        };
      };
    };
  };
}