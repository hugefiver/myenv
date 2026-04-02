#!/usr/bin/env bash
set -uo pipefail

POPUP_CLASSES=(
  "org.pulseaudio.pavucontrol"
  "nm-connection-editor"
  "org.gnome.Calculator"
  "gnome-calculator"
  "kcalc"
  "org.kde.kcalc"
  "org.kde.kcolorchooser"
  "org.gnome.font-viewer"
  "blueman-manager"
  ".blueman-manager-wrapped"
  "org.fcitx.fcitx5-config-qt"
  "fcitx5-config-qt"
)

is_popup() {
  local cls="$1"
  for p in "${POPUP_CLASSES[@]}"; do
    [[ "$cls" == "$p" ]] && return 0
  done
  return 1
}

SOCKET="${XDG_RUNTIME_DIR}/hypr/${HYPRLAND_INSTANCE_SIGNATURE}/.socket2.sock"
tracked_addr=""
grace_until=0

restore_input() {
  hyprctl --batch "keyword input:follow_mouse 1 ; keyword input:mouse_refocus true" 2>/dev/null || true
  tracked_addr=""
  grace_until=0
}

socat -u "UNIX-CONNECT:$SOCKET" - 2>/dev/null | while IFS= read -r event; do
  case "$event" in
    openwindow\>\>*)
      IFS=',' read -r ow_addr _ ow_class _ <<< "${event#openwindow>>}"

      if is_popup "$ow_class"; then
        tracked_addr="$ow_addr"
        grace_until=$((SECONDS + 2))
        hyprctl --batch "keyword input:follow_mouse 0 ; keyword input:mouse_refocus false" 2>/dev/null || true
      fi
      ;;

    activewindowv2\>\>*)
      new_addr="${event#activewindowv2>>}"

      if [[ -n "$tracked_addr" && "$new_addr" != "$tracked_addr" ]]; then
        [[ "$SECONDS" -lt "$grace_until" ]] && continue
        hyprctl dispatch closewindow "address:0x$tracked_addr" 2>/dev/null || true
        restore_input
      fi
      ;;

    closewindow\>\>*)
      cw_addr="${event#closewindow>>}"
      if [[ "$cw_addr" == "$tracked_addr" ]]; then
        restore_input
      fi
      ;;
  esac
done
