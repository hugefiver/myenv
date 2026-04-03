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

read -r VW VH < <(
  ffprobe -v error -select_streams v:0 \
    -show_entries stream=width,height -of csv=p=0 "$VIDEO" | tr ',' ' '
)
[ -z "$VW" ] && exit 1

TW=$EW
TH=$EH
if [ "$VW" -lt "$EW" ] || [ "$VH" -lt "$EH" ]; then
  if [ "$(( VW * EH ))" -lt "$(( VH * EW ))" ]; then
    TW=$VW
    TH=$(( VW * EH / EW ))
  else
    TH=$VH
    TW=$(( VH * EW / EH ))
  fi
  TW=$(( (TW / 2) * 2 ))
  TH=$(( (TH / 2) * 2 ))
fi

BASENAME=$(basename "$VIDEO")
CACHED="$CACHE_DIR/${BASENAME%.*}_${TW}x${TH}.mp4"

if [ ! -f "$CACHED" ] || [ "$VIDEO" -nt "$CACHED" ]; then
  VF="crop=ih*${TW}/${TH}:ih,scale=${TW}:${TH}:flags=bilinear"
  if [ "$(( VW * TH ))" -lt "$(( VH * TW ))" ]; then
    VF="crop=iw:iw*${TH}/${TW},scale=${TW}:${TH}:flags=bilinear"
  fi

  notify-send -t 8000 "壁纸" "正在转码 ${TW}×${TH} ..."
  if ffmpeg -y -hwaccel qsv -i "$VIDEO" \
    -vf "$VF" \
    -c:v h264_qsv -global_quality 20 -preset fast \
    -r 30 -an -movflags +faststart \
    "$CACHED" 2>/dev/null; then
    :
  elif ffmpeg -y -vaapi_device /dev/dri/renderD128 -i "$VIDEO" \
    -vf "$VF,format=nv12,hwupload" \
    -c:v h264_vaapi -qp 20 \
    -r 30 -an -movflags +faststart \
    "$CACHED" 2>/dev/null; then
    :
  else
    ffmpeg -y -i "$VIDEO" \
      -vf "$VF" \
      -c:v libx264 -profile:v high -tune fastdecode \
      -preset fast -crf 18 \
      -r 30 -an -movflags +faststart \
      "$CACHED" 2>/dev/null
  fi
  notify-send -t 3000 "壁纸" "转码完成"
fi

pkill mpvpaper 2>/dev/null || true
sleep 0.3
mpvpaper -fvs -o "no-audio loop panscan=1.0 hwdec=auto gpu-context=waylandvk cache=yes" "$OUTPUT" "$CACHED" >/dev/null 2>&1 &
