#!/usr/bin/env bash
# rofi 电源菜单 — 右上角小下拉栏

if pgrep -x rofi >/dev/null 2>&1; then
  pkill -x rofi
  exit 0
fi

entries="󰌾  锁屏\n󰍃  注销\n󰒲  休眠\n󰜉  重启\n󰐥  关机"

chosen=$(echo -e "$entries" | rofi -dmenu -theme ~/.config/rofi/power-menu.rasi -p "")

case "$chosen" in
  *锁屏*) hyprlock ;;
  *注销*) hyprctl dispatch exit ;;
  *休眠*) systemctl suspend ;;
  *重启*) systemctl reboot ;;
  *关机*) systemctl poweroff ;;
esac
