#!/usr/bin/env bash
set -uo pipefail

IMG="/tmp/hypr-cheatsheet.svg"

if pkill -f "swayimg.*hypr-cheatsheet" 2>/dev/null; then
  exit 0
fi

if [ ! -f "$IMG" ] || [ "$0" -nt "$IMG" ]; then
cat > "$IMG" << 'SVGEOF'
<svg xmlns="http://www.w3.org/2000/svg" width="1600" height="640">
<rect width="100%" height="100%" rx="16" fill="#1e1b18"/>
<style>
  .title { font: bold 22px "Noto Sans"; fill: #dba86b; }
  .hdr   { font: bold 14px "Noto Sans"; fill: #dba86b; }
  .key   { font: 13px "Noto Sans Mono", monospace; fill: #e6ddd4; }
</style>

<text x="36" y="36" class="title">⌨  Hyprland 快捷键速查</text>
<line x1="800" y1="56" x2="800" y2="608" stroke="#3a3530" stroke-width="1"/>

<!-- ══ 左栏 ══ -->
<text x="36" y="76" class="hdr">窗口管理</text>
<text class="key"><tspan x="36" y="98">Super + Return              终端</tspan>
<tspan x="36" dy="18">Super + K                   关闭窗口</tspan>
<tspan x="36" dy="18">Super + V                   浮动切换</tspan>
<tspan x="36" dy="18">Super + T                   切换分割方向</tspan>
<tspan x="36" dy="18">Super + Tab / Shift+Tab     下/上一窗口</tspan>
<tspan x="36" dy="18">Super + B/F/P/N 或 ←→↑↓     焦点 左/右/上/下</tspan>
<tspan x="36" dy="18">Super+Shift + 同上          移动窗口</tspan>
<tspan x="36" dy="18">Super+Alt + ←→↑↓            调整大小（按住连续）</tspan>
<tspan x="36" dy="18">Super + 鼠标左/右键拖动      移动 / 调整大小</tspan></text>

<text x="36" y="278" class="hdr">分组（标签页）</text>
<text class="key"><tspan x="36" y="300">Super + G                   创建/解散分组</tspan>
<tspan x="36" dy="18">Super + .  /  ,             下/上一标签</tspan>
<tspan x="36" dy="18">Super+Shift + G             移出分组</tspan></text>

<text x="36" y="378" class="hdr">布局</text>
<text class="key"><tspan x="36" y="400">Super + F9                  Master 方向轮转</tspan>
<tspan x="36" dy="18">Super + F10                 切换 Master 布局</tspan>
<tspan x="36" dy="18">Super + F11                 切换 Dwindle 布局</tspan>
<tspan x="36" dy="18">Super + F12                 交换主窗口</tspan></text>

<text x="36" y="496" class="hdr">跨屏</text>
<text class="key"><tspan x="36" y="518">Super+Ctrl + ↑↓             焦点切换屏幕</tspan>
<tspan x="36" dy="18">Super+Ctrl+Shift + ↑↓      移窗到另一屏</tspan></text>

<text x="36" y="576" class="hdr">前缀键  Super+X → …</text>
<text class="key"><tspan x="36" y="598">B 窗口列表  D 启动器  I Emacs  F 文件管理  K 关闭  L 锁屏  N 网络  C 剪贴板  H 速查  P 电源  R 重载  S 休眠  V 录屏  M 显示器</tspan></text>

<!-- ══ 右栏 ══ -->
<text x="820" y="76" class="hdr">工作区</text>
<text class="key"><tspan x="820" y="98">Super + 1~0                 切换工作区 1-10</tspan>
<tspan x="820" dy="18">Super+Shift + 1~0           移窗到工作区</tspan>
<tspan x="820" dy="18">Super + A/E 或 Ctrl+←→      上/下一工作区（当前屏）</tspan>
<tspan x="820" dy="18">Super+Shift + A/E / C-S-←→  移窗到上/下一工作区</tspan>
<tspan x="820" dy="18">Super + =                   新建空工作区（当前屏）</tspan>
<tspan x="820" dy="18">Super + D                   工作区总览 (Expo)</tspan>
<tspan x="820" dy="18">Super + S / Shift+S         特殊工作区 / 移到特殊</tspan></text>

<text x="820" y="248" class="hdr">启动器</text>
<text class="key"><tspan x="820" y="270">Super + Space                应用启动器</tspan>
<tspan x="820" dy="18">  ├ Ctrl+Enter              浮动模式启动</tspan>
<tspan x="820" dy="18">  └ Shift+Enter             空闲工作区启动</tspan>
<tspan x="820" dy="18">Super+Shift + Space / W     窗口列表</tspan>
<tspan x="820" dy="18">Super + I / M / C           Emacs / 文件管理 / 剪贴板</tspan></text>

<text x="820" y="400" class="hdr">截屏 / 录屏</text>
<text class="key"><tspan x="820" y="422">Print                       截图（区域选取）</tspan>
<tspan x="820" dy="18">Shift / Super / Ctrl + Print 当前窗口 / 全屏 / 标注</tspan>
<tspan x="820" dy="18">Super + R                   录屏开关</tspan></text>

<text x="820" y="510" class="hdr">系统</text>
<text class="key"><tspan x="820" y="532">Super + L                   锁屏</tspan>
<tspan x="820" dy="18">Super+Shift + Q / K / W     电源菜单 / 退出 / 网络</tspan>
<tspan x="820" dy="18">Super + \  /  /             顶栏显隐 / 本速查表</tspan></text>
</svg>
SVGEOF
fi

MON_W=1600
MON_H=900

if command -v hyprctl >/dev/null && command -v jq >/dev/null; then
  INFO=$(hyprctl monitors -j 2>/dev/null | jq -r '.[] | select(.focused) | "\(.width) \(.height)"' 2>/dev/null)
  if [ -n "$INFO" ]; then
    read -r MON_W MON_H <<< "$INFO"
  fi
fi

MON_W=${MON_W:-1600}
MON_H=${MON_H:-900}

MAX_W=$((MON_W * 75 / 100))
MAX_H=$((MON_H * 75 / 100))

TARGET_W=$MAX_W
TARGET_H=$((TARGET_W * 640 / 1600))

if [ "$TARGET_H" -gt "$MAX_H" ]; then
  TARGET_H=$MAX_H
  TARGET_W=$((TARGET_H * 1600 / 640))
fi

exec hyprctl dispatch exec "[float; size ${TARGET_W} ${TARGET_H}; center; pin] swayimg --class=hypr-cheatsheet \"$IMG\""
