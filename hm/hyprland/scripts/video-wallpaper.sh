#!/usr/bin/env bash
set -uo pipefail

VIDEO="${1:-}"
OUTPUT="${2:-DVI-I-1}"
CACHE_DIR="$HOME/.cache/video-wallpaper"

[ -z "$VIDEO" ] && exit 1
[ ! -f "$VIDEO" ] && exit 1

mkdir -p "$CACHE_DIR"

read -r PW PH SCALE < <(
  hyprctl monitors -j | jq -r --arg o "$OUTPUT" \
    '.[] | select(.name == $o) | "\(.width) \(.height) \(.scale)"'
)

[ -z "$PW" ] && exit 1

EW=$(jq -n "($PW / $SCALE) | floor | if . % 2 != 0 then . - 1 else . end")
EH=$(jq -n "($PH / $SCALE) | floor | if . % 2 != 0 then . - 1 else . end")

BASENAME=$(basename "$VIDEO")
CACHED="$CACHE_DIR/${BASENAME%.*}_${EW}x${EH}.mp4"

if [ ! -f "$CACHED" ] || [ "$VIDEO" -nt "$CACHED" ]; then
  read -r VW VH < <(
    ffprobe -v error -select_streams v:0 \
      -show_entries stream=width,height -of csv=p=0 "$VIDEO" | tr ',' ' '
  )

  VF="crop=ih*${EW}/${EH}:ih,scale=${EW}:${EH}:flags=lanczos"
  if [ "$(( VW * EH ))" -lt "$(( VH * EW ))" ]; then
    VF="crop=iw:iw*${EH}/${EW},scale=${EW}:${EH}:flags=lanczos"
  fi

  notify-send -t 8000 "壁纸" "正在转码 ${EW}×${EH} ..."
  ffmpeg -y -i "$VIDEO" \
    -vf "$VF" \
    -c:v libx264 -preset medium -crf 20 \
    -r 30 -an -movflags +faststart \
    "$CACHED" 2>/dev/null
  notify-send -t 3000 "壁纸" "转码完成"
fi

pkill mpvpaper 2>/dev/null || true
sleep 0.3
mpvpaper -fvs -o "no-audio loop panscan=1.0 hwdec=auto cache=yes" "$OUTPUT" "$CACHED" >/dev/null 2>&1 &
