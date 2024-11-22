{lib, ...}: {
  disko.devices = {
    disk.vda = {
      device = lib.mkDefault "/dev/vda";
      type = "disk";
      content = {
        type = "gpt";
        partitions = {
          MBR = {
            size = "1M";
            type = "EF02";
            priority = 1;
          };
          root = {
            name = "root";
            size = "100%";
            content = {
              type = "btrfs";
              extraArgs = ["-f"];
              mountpoint = "/";
              subvolumes = {
                "/swap" = {
                  mountpoint = "/swap";
                  mountOptions = [ "noatime" ];
                  swap.swapfile.size = "4G";
                };
              };
              swap."/swap/swapfile".size = "4G";
            };
          };
        };
      };
    };
  };
}
