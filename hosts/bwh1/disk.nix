{lib, ...}: {
  disko.devices = {
    disk.sda = {
      device = lib.mkDefault "/dev/sda";
      type = "disk";
      content = {
        type = "gpt";
        partitions = {
          boot = {
            size = "1M";
            type = "EF02";
          };
          root = {
            name = "root";
            size = "100%";
            content = {
              type = "btrfs";
              mountpoint = "/";
              subvolumes = {
                "/nix" = {
                  mountpoint = "/nix";
                  mountOptions = [ "noatime" ];
                };
                "/swap" = {
                  mountpoint = "/swap";
                  swap.swapfile.size = "1G";
                };
              };
              swap = {
                swapfile.size = "1G";
              };
            };
          };
        };
      };
    };
  };
}
