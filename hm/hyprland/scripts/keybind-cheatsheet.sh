#!/usr/bin/env bash
set -uo pipefail

IMG="/tmp/hypr-cheatsheet.svg"

if pkill -f "swayimg.*hypr-cheatsheet" 2>/dev/null; then
  exit 0
fi

if [ ! -f "$IMG" ] || [ "$0" -nt "$IMG" ]; then
cat > "$IMG" << 'SVGEOF'
<svg xmlns="http://www.w3.org/2000/svg" width="1400" height="820">
<rect width="100%" height="100%" rx="16" fill="#1e1b18"/>
<style>
  .title { font: bold 22px "Noto Sans"; fill: #dba86b; }
  .hdr   { font: bold 14px "Noto Sans"; fill: #dba86b; }
  .key   { font: 13px "Noto Sans Mono", monospace; fill: #e6ddd4; }
</style>

<text x="36" y="40" class="title">⌨  Hyprland 快捷键速查</text>

<!-- ══ 左栏 ══ -->
<text x="36" y="80" class="hdr">窗口管理</text>
<text class="key"><tspan x="36" y="102">Super + Return              终端</tspan>
<tspan x="36" dy="18">Super + K                   关闭窗口</tspan>
<tspan x="36" dy="18">Super + V                   浮动切换</tspan>
<tspan x="36" dy="18">Super + T                   切换分割方向</tspan>
<tspan x="36" dy="18">Super + Tab / Shift+Tab     下/上一窗口</tspan>
<tspan x="36" dy="18">Super + B/F/P/N 或 ←→↑↓     焦点 左/右/上/下</tspan>
<tspan x="36" dy="18">Super+Shift + 同上          移动窗口</tspan>
<tspan x="36" dy="18">Super+Alt + ←→↑↓            调整大小（按住连续）</tspan>
<tspan x="36" dy="18">Super + 鼠标左/右键拖动      移动 / 调整大小</tspan></text>

<text x="36" y="290" class="hdr">分组（标签页）</text>
<text class="key"><tspan x="36" y="312">Super + G                   创建/解散分组</tspan>
<tspan x="36" dy="18">Super + .  /  ,             下/上一标签</tspan>
<tspan x="36" dy="18">Super+Shift + G             移出分组</tspan></text>

<text x="36" y="386" class="hdr">工作区</text>
<text class="key"><tspan x="36" y="408">Super + 1~0                 切换工作区 1-10</tspan>
<tspan x="36" dy="18">Super+Shift + 1~0           移窗到工作区</tspan>
<tspan x="36" dy="18">Super + A/E 或 Ctrl+←→      上/下一工作区（当前屏）</tspan>
<tspan x="36" dy="18">Super+Shift + A/E / C-S-←→  移窗到上/下一工作区</tspan>
<tspan x="36" dy="18">Super + =                   新建空工作区（当前屏）</tspan>
<tspan x="36" dy="18">Super + D                   工作区总览 (Expo)</tspan>
<tspan x="36" dy="18">Super + S / Shift+S         特殊工作区 / 移到特殊</tspan></text>

<text x="36" y="560" class="hdr">跨屏</text>
<text class="key"><tspan x="36" y="582">Super+Ctrl + ↑↓             焦点切换屏幕</tspan>
<tspan x="36" dy="18">Super+Ctrl+Shift + ↑↓      移窗到另一屏</tspan></text>

<text x="36" y="640" class="hdr">布局</text>
<text class="key"><tspan x="36" y="662">Super + F9                  Master 方向轮转</tspan>
<tspan x="36" dy="18">Super + F10                 切换 Master 布局</tspan>
<tspan x="36" dy="18">Super + F11                 切换 Dwindle 布局</tspan>
<tspan x="36" dy="18">Super + F12                 交换主窗口</tspan></text>

<!-- ══ 右栏 ══ -->
<text x="720" y="80" class="hdr">启动器</text>
<text class="key"><tspan x="720" y="102">Super + Space                应用启动器</tspan>
<tspan x="720" dy="18">  ├ Ctrl+Enter              浮动模式启动</tspan>
<tspan x="720" dy="18">  └ Shift+Enter             空闲工作区启动</tspan>
<tspan x="720" dy="18">Super+Shift + Space / W     窗口列表</tspan>
<tspan x="720" dy="18">Super + I                   Emacs</tspan>
<tspan x="720" dy="18">Super + M                   文件管理器</tspan>
<tspan x="720" dy="18">Super + C                   剪贴板历史</tspan></text>

<text x="720" y="248" class="hdr">截屏 / 录屏</text>
<text class="key"><tspan x="720" y="270">Print                       截图（区域选取）</tspan>
<tspan x="720" dy="18">Shift + Print               截图（当前窗口）</tspan>
<tspan x="720" dy="18">Super + Print               截图（全屏）</tspan>
<tspan x="720" dy="18">Ctrl + Print                截图 + 标注编辑</tspan>
<tspan x="720" dy="18">Super + R                   录屏开关</tspan></text>

<text x="720" y="380" class="hdr">系统</text>
<text class="key"><tspan x="720" y="402">Super + L                   锁屏</tspan>
<tspan x="720" dy="18">Super+Shift + Q             电源菜单</tspan>
<tspan x="720" dy="18">Super+Shift + K             退出 Hyprland</tspan>
<tspan x="720" dy="18">Super+Shift + W             网络设置</tspan>
<tspan x="720" dy="18">Super + \                   顶栏显隐</tspan>
<tspan x="720" dy="18">Super + /                   本速查表</tspan></text>

<text x="720" y="520" class="hdr">前缀键  Super+X → …</text>
<text class="key"><tspan x="720" y="542">B 窗口列表   D 启动器   I Emacs</tspan>
<tspan x="720" dy="18">F 文件管理   K 关闭     L 锁屏</tspan>
<tspan x="720" dy="18">N 网络设置   C 剪贴板   H 快捷键</tspan>
<tspan x="720" dy="18">P 电源菜单   R 重载     S 休眠</tspan>
<tspan x="720" dy="18">V 录屏       M 显示器配置</tspan></text>

<line x1="700" y1="70" x2="700" y2="780" stroke="#3a3530" stroke-width="1"/>
</svg>
SVGEOF
fi

exec swayimg --class=hypr-cheatsheet "$IMG"
