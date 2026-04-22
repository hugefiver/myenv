#!/usr/bin/env bash
set -uo pipefail

run_once() {
  local pattern="$1"
  shift

  if ! pgrep -af "$pattern" >/dev/null 2>&1; then
    "$@" >/dev/null 2>&1 &
  fi
}

# NixOS UWSM 会话可能丢失 LOCALE_ARCHIVE，从系统环境补回
if [ -z "${LOCALE_ARCHIVE:-}" ] && [ -f /etc/set-environment ]; then
  eval "$(grep '^export LOCALE_ARCHIVE=' /etc/set-environment 2>/dev/null || true)"
fi

dbus-update-activation-environment --systemd --all >/dev/null 2>&1 || true

# kwalletd6：Hyprland 下提供 Secret Service D-Bus API（KDE 会话自带，这里显式启动）。
# kwallet-pam 已在登录时解锁，nm-applet 等应用直接用它存取 WiFi 密码。
run_once 'kwalletd[56]' kwalletd6

run_once '^hypridle$' hypridle
case "${BAR_BACKEND:-waybar}" in
  quickshell) run_once '^quickshell$' quickshell ;;
  *)          run_once '^waybar$' waybar ;;
esac
run_once '^dunst$' dunst
run_once '^hyprpolkitagent$' hyprpolkitagent
run_once 'clash-verge' clash-verge
# 备选：若 clash-verge 第一次启动 WebKit 渲染异常（界面缺样式），
# 把上面这行替换成下面的延迟启动块——等 systemd user manager
# 完整继承 Hyprland session env（含 WEBKIT_DISABLE_DMABUF_RENDERER 等）后再启。
# (
#   sleep 3
#   systemctl --user import-environment \
#       WAYLAND_DISPLAY XDG_SESSION_TYPE XDG_CURRENT_DESKTOP \
#       WEBKIT_DISABLE_DMABUF_RENDERER WEBKIT_DISABLE_COMPOSITING_MODE \
#       2>/dev/null || true
#   run_once 'clash-verge' clash-verge
# ) &

if ! pgrep -x awww-daemon >/dev/null 2>&1; then
  awww-daemon >/dev/null 2>&1 &
  sleep 0.4
fi

# 视频壁纸优先，回退到静态图片
_video="$HOME/Pictures/wallpapers/CH0273_home_260331013216_15f994_60.mp4"
if [ -f "$_video" ]; then
  ~/.config/hypr/scripts/video-wallpaper.sh "$_video" DVI-I-1 &
else
  for wallpaper in \
    "$HOME/Pictures/Wallpapers/default.png" \
    "$HOME/Pictures/Wallpapers/default.jpg" \
    "$HOME/Pictures/Wallpapers/default.jpeg"; do
    if [ -f "$wallpaper" ]; then
      awww img "$wallpaper" \
        --transition-type grow \
        --transition-pos center \
        --transition-duration 1.1 >/dev/null 2>&1 || true
      break
    fi
  done
fi

if ! pgrep -af 'wl-paste --type text --watch cliphist store' >/dev/null 2>&1; then
  wl-paste --type text --watch cliphist store >/dev/null 2>&1 &
fi

# DisplayLink 安全网：若 evdi 设备在 Hyprland 启动后才就绪，
# 延迟重载配置以触发 monitor 规则重新评估，之后重刷壁纸。
(
  sleep 1
  hyprctl dispatch dpms off
  sleep 0.3
  hyprctl dispatch dpms on
  sleep 0.5
  hyprctl reload
  sleep 0.5
  if ! pgrep -x mpvpaper >/dev/null 2>&1; then
    for wallpaper in \
      "$HOME/Pictures/Wallpapers/default.png" \
      "$HOME/Pictures/Wallpapers/default.jpg" \
      "$HOME/Pictures/Wallpapers/default.jpeg"; do
      if [ -f "$wallpaper" ]; then
        awww img "$wallpaper" --transition-type none 2>/dev/null || true
        break
      fi
    done
  fi
) >/dev/null 2>&1 &

if [ "${BAR_BACKEND:-waybar}" = "waybar" ]; then
  run_once 'waybar-autohide' ~/.config/hypr/scripts/waybar-autohide.sh
fi
