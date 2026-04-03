#!/usr/bin/env bash
set -uo pipefail

EDGE_PX=6
BAR_PX=48
FLASH_SECS=2
POLL_SEC=0.3

STATE_FILE="/tmp/waybar-autohide-state"
PINNED_FILE="/tmp/waybar-autohide-pinned"
FLASH_FILE="/tmp/waybar-autohide-flash"

rm -f "$PINNED_FILE" "$FLASH_FILE"
echo "visible" > "$STATE_FILE"

# NixOS 的 waybar 被 makeWrapper 包装，实际进程名是 .waybar-wrapped
bar_signal() { pkill -SIGUSR1 waybar 2>/dev/null || true; }

bar_show() {
  [[ "$(<"$STATE_FILE")" == "visible" ]] && return
  bar_signal
  echo "visible" > "$STATE_FILE"
}

bar_hide() {
  [[ "$(<"$STATE_FILE")" == "hidden" ]] && return
  bar_signal
  echo "hidden" > "$STATE_FILE"
}

# 判断光标是否在任一显示器的顶边附近（相对该显示器 y 偏移）
cursor_near_top() {
  local threshold=$1
  local pos cx cy
  pos=$(hyprctl cursorpos 2>/dev/null) || return 1
  [[ "$pos" =~ ([0-9]+),\ *([0-9]+) ]] || return 1
  cx=${BASH_REMATCH[1]}; cy=${BASH_REMATCH[2]}

  local tops
  tops=$(hyprctl monitors -j 2>/dev/null | jq -r '.[] | "\(.x) \(.y) \(.width) \(.height)"') || return 1

  while IFS=' ' read -r mx my mw mh; do
    # 光标 x 在此显示器范围内，且 y 在顶边 threshold 像素内
    if (( cx >= mx && cx < mx + mw && cy >= my && cy < my + threshold )); then
      return 0
    fi
  done <<< "$tops"
  return 1
}

toggle_pin() {
  if [[ -f "$PINNED_FILE" ]]; then
    rm -f "$PINNED_FILE"
    bar_hide
  else
    touch "$PINNED_FILE"
    bar_show
  fi
}

trap toggle_pin USR1
trap 'kill $(jobs -p) 2>/dev/null; rm -f "$STATE_FILE" "$PINNED_FILE" "$FLASH_FILE"; exit 0' EXIT INT TERM

# 等 waybar 完全就绪再发信号
for _i in $(seq 1 30); do
  pgrep waybar >/dev/null 2>&1 && break
  sleep 0.2
done
sleep 1.5
# 默认显示，不自动隐藏（用户按 Super+\ 可切换 pin）

(
  socat -u "UNIX-CONNECT:${XDG_RUNTIME_DIR}/hypr/${HYPRLAND_INSTANCE_SIGNATURE}/.socket2.sock" - 2>/dev/null | while IFS= read -r event; do
    case "$event" in workspace*|focusedmon*) touch "$FLASH_FILE" ;; esac
  done
) &

while true; do
  sleep "$POLL_SEC" &
  wait $!

  [[ -f "$PINNED_FILE" ]] && continue

  cur_state=$(<"$STATE_FILE")

  if [[ -f "$FLASH_FILE" ]]; then
    rm -f "$FLASH_FILE"
    bar_show
    sleep "$FLASH_SECS" &
    wait $!
    [[ -f "$PINNED_FILE" ]] && continue
    cursor_near_top "$BAR_PX" || bar_hide
    continue
  fi

  if [[ "$cur_state" == "visible" ]]; then
    cursor_near_top "$BAR_PX" || bar_hide
  else
    cursor_near_top "$EDGE_PX" && bar_show
  fi
done
