#!/usr/bin/env bash
set -euo pipefail

run_once() {
  local pattern="$1"
  shift

  if ! pgrep -af "$pattern" >/dev/null 2>&1; then
    "$@" >/dev/null 2>&1 &
  fi
}

dbus-update-activation-environment --systemd --all >/dev/null 2>&1 || true

run_once '^waybar$' waybar
run_once '^dunst$' dunst
run_once '^hyprpolkitagent$' hyprpolkitagent
run_once '^nm-applet$' nm-applet --indicator

if ! pgrep -x swww-daemon >/dev/null 2>&1; then
  swww-daemon >/dev/null 2>&1 &
  sleep 0.4
fi

for wallpaper in \
  "$HOME/Pictures/Wallpapers/default.png" \
  "$HOME/Pictures/Wallpapers/default.jpg" \
  "$HOME/Pictures/Wallpapers/default.jpeg"; do
  if [ -f "$wallpaper" ]; then
    swww img "$wallpaper" \
      --transition-type grow \
      --transition-pos center \
      --transition-duration 1.1 >/dev/null 2>&1 || true
    break
  fi
done

if ! pgrep -af 'wl-paste --type text --watch cliphist store' >/dev/null 2>&1; then
  wl-paste --type text --watch cliphist store >/dev/null 2>&1 &
fi
