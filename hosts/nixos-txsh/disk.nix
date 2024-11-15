{lib, ...}: {
  disko.devices = {
    disk.vda = {
      device = lib.mkDefault "/dev/vda";
      type = "disk";
      content = {
        type = "gpt";
        partitions = {
          MBR = { size = "1M"; type = "EF02"; priority = 1; };
          root = { name = "root"; size = "100%"; content = { type = "filesystem"; format = "btrfs"; mountpoint = "/"; }; };
        };
      };
    };
  };
}