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
# 不加 -x 做子串匹配，同时匹配 "waybar" 和 ".waybar-wrapped"
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

cursor_y() {
  local pos
  pos=$(hyprctl cursorpos 2>/dev/null) || { echo 9999; return; }
  if [[ "$pos" =~ ,\ *([0-9]+) ]]; then
    echo "${BASH_REMATCH[1]}"
  else
    echo 9999
  fi
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
bar_hide

(
  socat -u "UNIX-CONNECT:${XDG_RUNTIME_DIR}/hypr/${HYPRLAND_INSTANCE_SIGNATURE}/.socket2.sock" - 2>/dev/null | while IFS= read -r event; do
    case "$event" in workspace*|focusedmon*) touch "$FLASH_FILE" ;; esac
  done
) &

while true; do
  sleep "$POLL_SEC" &
  wait $!

  [[ -f "$PINNED_FILE" ]] && continue

  y=$(cursor_y)
  cur_state=$(<"$STATE_FILE")

  if [[ -f "$FLASH_FILE" ]]; then
    rm -f "$FLASH_FILE"
    bar_show
    sleep "$FLASH_SECS" &
    wait $!
    [[ -f "$PINNED_FILE" ]] && continue
    y=$(cursor_y)
    (( y <= BAR_PX )) || bar_hide
    continue
  fi

  if [[ "$cur_state" == "visible" ]]; then
    (( y <= BAR_PX )) || bar_hide
  else
    (( y <= EDGE_PX )) && bar_show
  fi
done
