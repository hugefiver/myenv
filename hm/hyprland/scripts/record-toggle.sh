#!/usr/bin/env bash
# 录屏切换：运行中则停止，否则用 slurp 选区后开始录屏。
set -euo pipefail

VIDEOS_DIR="${XDG_VIDEOS_DIR:-$HOME/Videos}"
mkdir -p "$VIDEOS_DIR"

if pgrep -x wf-recorder > /dev/null 2>&1; then
  pkill -INT wf-recorder
  notify-send -u normal -i camera-video "录屏已停止" "视频已保存到 $VIDEOS_DIR"
else
  GEOMETRY="$(slurp 2>/dev/null)" || exit 0  # 用户取消选区则直接退出
  FILENAME="$VIDEOS_DIR/recording_$(date +%Y%m%d_%H%M%S).mp4"
  notify-send -u normal -i camera-video "开始录屏" "再次按下快捷键停止录屏"
  wf-recorder -g "$GEOMETRY" -f "$FILENAME" &
fi
