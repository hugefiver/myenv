{lib, ...}: {
  disko.devices = {
    disk.vda = {
      device = lib.mkDefault "/dev/vda";
      type = "disk";
      content = {
        type = "gpt";
        partitions = {
          boot = {
            size = "1M";
            type = "EF02";
            priority = 1;
          };
          ESP = {
            size = "200M";
            type = "EF00";
            start = "2M";
            priority = 2;
            content = {
              type = "filesystem";
              format = "vfat";
              mountpoint = "/boot/efi";
              mountOptions = ["umask=0077"];
            };
          };
          root = {
            name = "root";
            size = "100%";
            priority = 3;
            content = {
              type = "btrfs";
              extraArgs = ["-f"];
              mountpoint = "/";
              subvolumes = {
                "/boot" = {
                  mountpoint = "/boot";
                };
                "/nix" = {
                  mountpoint = "/nix";
                  mountOptions = ["noatime"];
                };
                "/swap" = {
                  mountpoint = "/swap";
                  swap.swapfile.size = "1G";
                };
              };
            };
          };
        };
      };
    };
  };
}
