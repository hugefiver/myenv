#!/usr/bin/env bash

STATE="/tmp/rofi-launcher-mode"

if pgrep -x rofi >/dev/null 2>&1; then
  pkill -x rofi
  sleep 0.05
  current=$(<"$STATE" 2>/dev/null) || current=0
  next=$(( (current + 1) % 3 ))
else
  next=0
fi

echo "$next" > "$STATE"

case "$next" in
  0) exec rofi -show drun -show-icons ;;
  1)
    xbel="$HOME/.local/share/recently-used.xbel"
    if [[ ! -f "$xbel" ]]; then
      notify-send "最近文件" "暂无记录"
      exit 0
    fi
    selected=$(grep 'href="file://' "$xbel" | \
      sed 's/.*href="file:\/\///; s/".*//' | \
      while IFS= read -r url; do printf '%b\n' "${url//%/\\x}"; done | \
      tac | awk '!seen[$0]++' | head -80 | \
      while IFS= read -r path; do [[ -e "$path" ]] && echo "$path"; done | \
      rofi -dmenu -i -p "📁 最近文件") || exit 0
    [[ -n "$selected" ]] && xdg-open "$selected" &
    ;;
  2) exec rofi -show filebrowser -show-icons ;;
esac
