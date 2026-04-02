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

# kwalletd6：Hyprland 下提供 Secret Service D-Bus API（KDE 会话自带，这里显式启动）。
# kwallet-pam 已在登录时解锁，nm-applet 等应用直接用它存取 WiFi 密码。
run_once 'kwalletd[56]' kwalletd6

run_once '^hypridle$' hypridle
run_once '^waybar$' waybar
run_once '^dunst$' dunst
run_once '^hyprpolkitagent$' hyprpolkitagent

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

# DisplayLink 安全网：若 evdi 设备在 Hyprland 启动后才就绪，
# 延迟重载配置以触发 monitor 规则重新评估，之后再自动缩放。
(sleep 1 && hyprctl reload && sleep 0.5 && ~/.config/hypr/scripts/auto-scale.sh) >/dev/null 2>&1 &
