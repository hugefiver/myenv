#!/usr/bin/env bash
set -uo pipefail

fmt() {
  local key="$1" act="$2"
  printf '<tt>  %-28s %s</tt>' "$key" "$act"
}

hdr() {
  printf '<b><span size="x-large">%s</span></b>' "$1"
}

{
hdr "窗口管理"
fmt "&lt;Super&gt; Return"             "终端"
fmt "&lt;Super&gt; K"                  "关闭窗口"
fmt "&lt;Super&gt; V"                  "浮动切换"
fmt "&lt;Super&gt; T"                  "切换分割方向"
fmt "&lt;Super&gt; Tab / Shift+Tab"    "下/上一窗口"
fmt "&lt;Super&gt; B/F/P/N 或 ←→↑↓"   "焦点 左/右/上/下"
fmt "&lt;Super+Shift&gt; B/F/P/N/←→↑↓" "移窗 左/右/上/下"
fmt "&lt;Super+Alt&gt; ←→↑↓"          "调整窗口大小（按住连续）"
fmt "&lt;Super&gt; + 鼠标左键拖动"     "移动窗口"
fmt "&lt;Super&gt; + 鼠标右键拖动"     "调整大小"
echo ""
hdr "分组（标签页）"
fmt "&lt;Super&gt; G"                  "创建/解散分组"
fmt "&lt;Super&gt; . / ,"              "下/上一标签"
fmt "&lt;Super+Shift&gt; G"            "移出分组"
echo ""
hdr "工作区"
fmt "&lt;Super&gt; 1~0"                "切换工作区 1-10"
fmt "&lt;Super+Shift&gt; 1~0"          "移窗到工作区"
fmt "&lt;Super&gt; A/E 或 Ctrl+←→"    "上/下一工作区（当前屏）"
fmt "&lt;Super+Shift&gt; A/E / C-S-←→" "移窗到上/下一工作区"
fmt "&lt;Super&gt; ="                  "新建空工作区（当前屏）"
fmt "&lt;Super&gt; D"                  "工作区总览 (Expo)"
fmt "&lt;Super&gt; S"                  "特殊工作区"
fmt "&lt;Super+Shift&gt; S"            "移到特殊工作区"
echo ""
hdr "跨屏"
fmt "&lt;Super+Ctrl&gt; A/E 或 ↑↓"    "焦点切换屏幕"
fmt "&lt;Super+C+S&gt; A/E 或 ↑↓"     "移窗到另一屏"
echo ""
hdr "启动器"
fmt "&lt;Super&gt; Space"              "应用启动器"
fmt "  ├ Ctrl+Enter"                  "  浮动模式启动"
fmt "  └ Shift+Enter"                 "  空闲工作区启动"
fmt "&lt;Super+Shift&gt; Space / W"    "窗口列表"
fmt "&lt;Super&gt; I"                  "Emacs"
fmt "&lt;Super&gt; M"                  "文件管理器"
fmt "&lt;Super&gt; C"                  "剪贴板历史"
echo ""
hdr "截屏 / 录屏"
fmt "Print"                          "截图（区域选取）"
fmt "Shift + Print"                  "截图（当前窗口）"
fmt "&lt;Super&gt; Print"              "截图（全屏）"
fmt "Ctrl + Print"                   "截图 + 标注编辑"
fmt "&lt;Super&gt; R"                  "录屏开关"
echo ""
hdr "布局"
fmt "&lt;Super&gt; F9"                 "Master 方向轮转"
fmt "&lt;Super&gt; F10"                "切换 Master 布局"
fmt "&lt;Super&gt; F11"                "切换 Dwindle 布局"
fmt "&lt;Super&gt; F12"                "交换主窗口"
echo ""
hdr "系统"
fmt "&lt;Super&gt; L"                  "锁屏"
fmt "&lt;Super+Shift&gt; Q"            "电源菜单"
fmt "&lt;Super+Shift&gt; K"            "退出 Hyprland"
fmt "&lt;Super+Shift&gt; W"            "网络设置"
fmt "&lt;Super&gt; \\"                 "顶栏显隐"
fmt "&lt;Super&gt; /"                  "本速查表"
echo ""
hdr "前缀键  &lt;Super&gt; X → ..."
fmt "B 窗口列表    D 启动器"         "I Emacs"
fmt "F 文件管理    K 关闭"           "L 锁屏"
fmt "N 网络设置    C 剪贴板"         "H 快捷键"
fmt "P 电源菜单    R 重载配置"       "S 休眠  V 录屏"
} | rofi -dmenu -markup-rows -i -p "" -no-custom \
  -theme-str '
* {
  bg: #eb1e1b18;
  fg: #e6ddd4;
  accent: #dba86b;
  font: "Noto Sans 12";
}
window {
  width: 52%;
  border: 1px solid #1fdbaf6e;
  border-radius: 16px;
  background-color: @bg;
}
mainbox {
  background-color: transparent;
  padding: 24px 28px;
}
inputbar { enabled: false; }
listview {
  lines: 50;
  columns: 1;
  scrollbar: false;
  fixed-height: false;
  background-color: transparent;
  spacing: 1;
}
element {
  padding: 4px 12px;
  background-color: transparent;
  text-color: @fg;
}
element selected {
  background-color: #1fdba86b;
  border-radius: 8px;
}
element-text {
  text-color: inherit;
  highlight: none;
}
'
