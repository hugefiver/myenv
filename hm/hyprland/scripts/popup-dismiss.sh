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
declare -A tracked=()
grace_until=0

restore_input() {
  hyprctl --batch "keyword input:follow_mouse 1 ; keyword input:mouse_refocus true" 2>/dev/null || true
  tracked=()
  grace_until=0
}

close_all_tracked() {
  for addr in "${!tracked[@]}"; do
    hyprctl dispatch closewindow "address:0x$addr" 2>/dev/null || true
  done
  restore_input
}

socat -u "UNIX-CONNECT:$SOCKET" - 2>/dev/null | while IFS= read -r event; do
  case "$event" in
    openwindow\>\>*)
      IFS=',' read -r ow_addr _ ow_class _ <<< "${event#openwindow>>}"

      if is_popup "$ow_class"; then
        tracked["$ow_addr"]=1
        grace_until=$((SECONDS + 2))
        hyprctl --batch "keyword input:follow_mouse 0 ; keyword input:mouse_refocus false" 2>/dev/null || true
      fi
      ;;

    activewindowv2\>\>*)
      new_addr="${event#activewindowv2>>}"

      [[ ${#tracked[@]} -eq 0 ]] && continue
      [[ -n "${tracked[$new_addr]+x}" ]] && continue
      [[ "$SECONDS" -lt "$grace_until" ]] && continue

      new_class=$(hyprctl activewindow -j 2>/dev/null | jq -r '.class // ""')
      if is_popup "$new_class"; then
        tracked["$new_addr"]=1
        continue
      fi

      close_all_tracked
      ;;

    closewindow\>\>*)
      cw_addr="${event#closewindow>>}"
      unset "tracked[$cw_addr]" 2>/dev/null || true
      [[ ${#tracked[@]} -eq 0 ]] && restore_input
      ;;
  esac
done
