#!/usr/bin/env bash
set -euo pipefail

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
)

is_popup() {
  for p in "${POPUP_CLASSES[@]}"; do
    [ "$1" = "$p" ] && return 0
  done
  return 1
}

SOCKET="$XDG_RUNTIME_DIR/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock"
tracked_addr=""

restore_follow_mouse() {
  if [ -n "$tracked_addr" ]; then
    hyprctl keyword input:follow_mouse 1 2>/dev/null || true
    tracked_addr=""
  fi
}

socat -u "UNIX-CONNECT:$SOCKET" - | while IFS= read -r event; do
  case "$event" in
    openwindow\>\>*)
      rest="${event#openwindow>>}"
      ow_addr="${rest%%,*}"
      rest="${rest#*,}"; rest="${rest#*,}"
      ow_class="${rest%%,*}"

      if is_popup "$ow_class"; then
        tracked_addr="$ow_addr"
        hyprctl keyword input:follow_mouse 0 2>/dev/null || true
      fi
      ;;

    activewindowv2\>\>*)
      new_addr="${event#activewindowv2>>}"

      if [ -n "$tracked_addr" ] && [ "$new_addr" != "$tracked_addr" ]; then
        hyprctl dispatch closewindow "address:0x$tracked_addr" 2>/dev/null || true
        restore_follow_mouse
      fi
      ;;

    closewindow\>\>*)
      cw_addr="${event#closewindow>>}"
      if [ "$cw_addr" = "$tracked_addr" ]; then
        restore_follow_mouse
      fi
      ;;
  esac
done
