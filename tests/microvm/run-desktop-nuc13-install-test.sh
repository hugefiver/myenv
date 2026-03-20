#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
WORKDIR="/tmp/desktop-nuc13-microvm-test"
RESULTS_DIR="$WORKDIR/results"
KEY_PATH="$WORKDIR/desktop-nuc13-test-key"
SSH_CMD=(ssh -F /dev/null -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -i "$KEY_PATH" -p 2222 root@127.0.0.1)
INSTALLER_PID=""

cleanup() {
  if [[ -n "$INSTALLER_PID" ]] && kill -0 "$INSTALLER_PID" 2>/dev/null; then
    kill "$INSTALLER_PID" 2>/dev/null || true
    wait "$INSTALLER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

mkdir -p "$WORKDIR" "$RESULTS_DIR"
rm -f "$RESULTS_DIR"/*
rm -f \
  "$WORKDIR/desktop-nuc13-cache.img" \
  "$WORKDIR/desktop-nuc13-main.img" \
  "$WORKDIR/desktop-nuc13-installer-vars.fd" \
  "$WORKDIR/desktop-nuc13-installer.sock" \
  "$KEY_PATH" \
  "$KEY_PATH.pub"

ssh-keygen -t ed25519 -N "" -f "$KEY_PATH" -C "desktop-nuc13-test" >/dev/null

echo "[1/4] Building disko verification microvm runner"
RUNNER_PATH=$(cd "$ROOT_DIR" && DESKTOP_NUC13_TEST_PUBKEY_PATH="$KEY_PATH.pub" nix build --impure --no-link --print-out-paths .#desktop-nuc13-installer-vm)

echo "[2/4] Starting verification microvm"
pushd "$WORKDIR" >/dev/null
"$RUNNER_PATH/bin/microvm-run" >"$RESULTS_DIR/installer-console.log" 2>&1 &
INSTALLER_PID=$!
popd >/dev/null

echo "[3/4] Waiting for SSH in guest"
for _ in $(seq 1 180); do
  if "${SSH_CMD[@]}" true >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! "${SSH_CMD[@]}" true >/dev/null 2>&1; then
  echo "Guest SSH did not become ready" >&2
  exit 1
fi

echo "[4/4] Running disko verification via SSH"
FORMAT_SCRIPT=$(cd "$ROOT_DIR" && nix build --impure --no-link --print-out-paths --expr '
  let
    flake = builtins.getFlake (toString ./.);
    sys = flake.inputs.nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = {
        diskoTestSkipSwapFiles = true;
      };
      modules = [
        flake.inputs.disko.nixosModules.disko
        (flake.outPath + "/desktops/nuc13/disk.nix")
        ({ lib, ... }: {
          networking.hostName = "desktop-nuc13-disko-test";
          system.stateVersion = "25.11";
          disko.rootMountPoint = "/mnt/target";
          disko.devices.disk.nvme0.device = lib.mkForce "/dev/nvme0n1";
          disko.devices.disk.hdd0.device = lib.mkForce "/dev/sda";
        })
      ];
    };
  in sys.config.system.build.formatScript
')

MOUNT_SCRIPT=$(cd "$ROOT_DIR" && nix build --impure --no-link --print-out-paths --expr '
  let
    flake = builtins.getFlake (toString ./.);
    sys = flake.inputs.nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      specialArgs = {
        diskoTestSkipSwapFiles = true;
      };
      modules = [
        flake.inputs.disko.nixosModules.disko
        (flake.outPath + "/desktops/nuc13/disk.nix")
        ({ lib, ... }: {
          networking.hostName = "desktop-nuc13-disko-test";
          system.stateVersion = "25.11";
          disko.rootMountPoint = "/mnt/target";
          disko.devices.disk.nvme0.device = lib.mkForce "/dev/nvme0n1";
          disko.devices.disk.hdd0.device = lib.mkForce "/dev/sda";
        })
      ];
    };
  in sys.config.system.build.mountScript
')

cat > "$WORKDIR/verify-remote.sh" <<EOF
#!/usr/bin/env bash
set -euxo pipefail
exec > >(tee /mnt/results/disko-verify.log) 2>&1
lsblk
test -b /dev/nvme0n1
test -b /dev/sda
mkdir -p /mnt/target
 "$FORMAT_SCRIPT"
 "$MOUNT_SCRIPT"
lsblk > /mnt/results/lsblk-after-install.txt
blkid > /mnt/results/blkid-after-install.txt
pvs > /mnt/results/pvs-after-install.txt
vgs > /mnt/results/vgs-after-install.txt
lvs -a -o+devices,segtype,cache_mode > /mnt/results/lvs-after-install.txt
findmnt -R /mnt/target > /mnt/results/findmnt-after-mount.txt
btrfs subvolume list /mnt/target > /mnt/results/btrfs-subvolumes.txt
ls -la /mnt/target/swap > /mnt/results/swap-dir.txt
grep -q '/mnt/target/boot' /mnt/results/findmnt-after-mount.txt
grep -q 'cache' /mnt/results/lvs-after-install.txt
grep -q '@root' /mnt/results/btrfs-subvolumes.txt
grep -q '@nix' /mnt/results/btrfs-subvolumes.txt
grep -q '@swap' /mnt/results/btrfs-subvolumes.txt
grep -q '@log' /mnt/results/btrfs-subvolumes.txt
test ! -e /mnt/target/swap/swapfile
touch /mnt/results/disko-verify-success
EOF
chmod +x "$WORKDIR/verify-remote.sh"

scp -F /dev/null -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -i "$KEY_PATH" -P 2222 "$WORKDIR/verify-remote.sh" root@127.0.0.1:/root/verify-remote.sh
"${SSH_CMD[@]}" /root/verify-remote.sh

echo "Stopping verification microvm"
cleanup
INSTALLER_PID=""

echo "Verification artifacts are in: $RESULTS_DIR"
