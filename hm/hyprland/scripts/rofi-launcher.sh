#!/usr/bin/env bash
set -uo pipefail

STATE="/tmp/rofi-launcher-mode"
CMD_FILE="/tmp/rofi-launcher-cmd"

# ── 循环切换模式 ──────────────────────────────────────────────
if pgrep -x rofi >/dev/null 2>&1; then
  pkill -x rofi
  sleep 0.05
  current=$(<"$STATE" 2>/dev/null) || current=0
  next=$(( (current + 1) % 3 ))
else
  next=0
fi

echo "$next" > "$STATE"

# ── 找到下一个空闲工作区 (1-10 中未被占用的最小值) ─────────────
next_free_ws() {
  local used
  used=$(hyprctl workspaces -j 2>/dev/null | jq -r '.[].id' | sort -n)
  for i in $(seq 1 10); do
    if ! echo "$used" | grep -qx "$i"; then
      echo "$i"
      return
    fi
  done
  echo "11"  # 全满则用 11
}

# ── drun 启动并根据快捷键决定窗口行为 ─────────────────────────
#   Enter        → 正常平铺
#   Ctrl+Enter   → 浮动模式
#   Shift+Enter  → 下一个空闲工作区
launch_drun() {
  rm -f "$CMD_FILE"

  rofi -show drun -show-icons \
    -drun-run-command "echo {cmd} > $CMD_FILE" \
    -kb-accept-custom '' \
    -kb-accept-alt '' \
    -kb-custom-1 'Control+Return' \
    -kb-custom-2 'Shift+Return'
  local rc=$?

  local cmd
  cmd=$(<"$CMD_FILE" 2>/dev/null) || return
  rm -f "$CMD_FILE"
  [[ -z "$cmd" ]] && return

  case $rc in
    0)  # 正常启动
        eval "$cmd" &
        ;;
    10) # 浮动
        hyprctl dispatch exec "[float; size 50% 60%; center]" -- "$cmd"
        ;;
    11) # 空闲工作区
        local ws
        ws=$(next_free_ws)
        hyprctl dispatch exec "[workspace $ws]" -- "$cmd"
        ;;
  esac
}

case "$next" in
  0) launch_drun ;;
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
