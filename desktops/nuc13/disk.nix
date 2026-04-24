args@{ lib, ... }:
let
  diskoTestSkipSwapFiles = args.diskoTestSkipSwapFiles or false;
  createSwapfile = !diskoTestSkipSwapFiles;
in {
  disko.devices = {
    disk = {
      nvme0 = {
        device = lib.mkDefault "/dev/nvme0n1";
        type = "disk";
        content = {
          type = "gpt";
          partitions = {
            ESP = {
              size = "400M";
              type = "EF00";
              content = {
                type = "filesystem";
                format = "vfat";
                mountpoint = "/boot";
                mountOptions = [ "umask=0077" ];
              };
            };
            lvm_cache = {
              size = "100%";
            };
          };
        };
      };

      hdd0 = {
        device = lib.mkDefault "/dev/sda";
        type = "disk";
        content = {
          type = "gpt";
          partitions = {
            lvm_main = {
              size = "100%";
              content = {
                type = "lvm_pv";
                vg = "pool";
              };
            };
          };
        };
      };
    };

    lvm_vg = {
      pool = {
        type = "lvm_vg";
        lvs = {
          root = {
            size = "100%FREE";
            content = {
              type = "btrfs";
              preCreateHook = ''
                if ! pvs /dev/nvme0n1p2 >/dev/null 2>&1; then
                  pvcreate -ff -y /dev/nvme0n1p2
                fi

                if ! pvs --noheadings -o vg_name /dev/nvme0n1p2 2>/dev/null | tr -d '[:space:]' | grep -qx "pool"; then
                  vgextend pool /dev/nvme0n1p2
                fi

                root_segtype="$(lvs --noheadings -o segtype pool/root | tr -d '[:space:]')"
                if [ "$root_segtype" != "cache" ]; then
                  if lvs pool/cache_data >/dev/null 2>&1 && ! lvs pool/cache_meta >/dev/null 2>&1; then
                    lvremove --yes --force pool/cache_data
                  fi

                  if ! lvs pool/cache_meta >/dev/null 2>&1; then
                    lvcreate --yes -L 128M -n cache_meta pool /dev/nvme0n1p2
                  fi

                  if ! lvs pool/cache_data >/dev/null 2>&1; then
                    lvcreate --yes -l 100%FREE -n cache_data pool /dev/nvme0n1p2
                  fi

                  cache_segtype="$(lvs --noheadings -o segtype pool/cache_data 2>/dev/null | tr -d '[:space:]')"
                  if [ "$cache_segtype" != "cache-pool" ]; then
                    lvconvert --yes --type cache-pool --poolmetadataspare n pool/cache_data --poolmetadata pool/cache_meta
                  fi

                  lvconvert --yes --type cache --cachepool pool/cache_data --cachemode writeback pool/root
                fi
              '';
              extraArgs = [ "-f" ];
              subvolumes = {
                "@root" = {
                  mountpoint = "/";
                  mountOptions = [ "user_subvol_rm_allowed" ];
                };
                "@nix" = {
                  mountpoint = "/nix";
                  mountOptions = [ "noatime" "compress=zstd" ];
                };
                "@swap" = {
                  mountpoint = "/swap";
                  mountOptions = [ "noatime" "nodatacow" ];
                } // lib.optionalAttrs createSwapfile {
                  swap.swapfile.size = "16G";
                };
                "@log" = {
                  mountpoint = "/var/log";
                  mountOptions = [ "nodatacow" ];
                };
              };
            };
          };
        };
      };
    };
  };
}
