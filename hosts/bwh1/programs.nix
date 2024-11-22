{
  pkgs,
  unstable,
  ...
}: {
  services.xray = { enable = true; settingsFile = "/etc/xray/config.json"; };

  systemd.services.rclone-mount = {
    enable = true;
    description = "Mount Rclone driver when starup.";

    wantedBy = ["multi-user.target"];
    after = ["network-online.target"];
    requires = ["network-online.target"];

    unitConfig.Type = "forking";

    preStart = "/run/current-system/sw/bin/mkdir -p /mnt/hath";
    serviceConfig.ExecStart = ''
      ${unstable.rclone}/bin/rclone mount \
      --config /root/.config/rclone/rclone.conf \
      cloud:/hath /mnt/hath \
      --vfs-read-chunk-size 10M --vfs-read-chunk-size-limit 200M \
      --cache-dir /tmp/hath-cache --vfs-cache-mode full \
      --vfs-cache-mode full \
      --vfs-cache-max-size 15G --vfs-cache-max-age 72h \
      --no-checksum --no-modtime \
      --transfers 16
    '';
  };

  systemd.services.mount-hath = {
    enable = true;
    wantedBy = ["multi-user.target"];
    after = ["rclone-mount.service"];
    requires = ["rclone-mount.service"];

    unitConfig.Type = "simple";

    preStart = "${pkgs.bash}/bin/bash -c 'sleep 10; mkdir -p /root/hath/{cache,true,download}'";
    serviceConfig = {
      RemainAfterExit = true;
      ExecStart = "${pkgs.writeShellScript "mount_hath.sh" ''
        #!/run/current-system/sw/bin/bash
        for dir in "cache" "true" "download"; do
          /run/current-system/sw/bin/mount --bind /mnt/hath/''${dir} /root/hath/''${dir};
        done
      ''}";
      ExecStop = "${pkgs.bash}/bin/bash -c 'umount /root/hath/{cache,true,download}'";
    };
  };

  systemd.services.start-hath = {
    enable = true;
    wantedBy = ["multi-user.target"];
    after = ["mount-hath.service"];
    requires = ["mount-hath.service"];
    restartIfChanged = false;

    unitConfig.Type = "oneshot";

    serviceConfig = {
      ExecStart = ''
        ${pkgs.tmux}/bin/tmux new-session -d -s hath -c /root/hath java -jar HentaiAtHome.jar
      '';
    };
  };
}
