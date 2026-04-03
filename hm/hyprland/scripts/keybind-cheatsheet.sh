#!/usr/bin/env bash
set -uo pipefail

CONF="${XDG_CONFIG_HOME:-$HOME/.config}/hypr/keybinds.conf"
[[ -f "$CONF" ]] || { notify-send "Keybind Cheatsheet" "keybinds.conf not found"; exit 1; }

declare -A raw=()
declare -a order=()
in_submap=""
ws_mods="" move_ws_mods="" nav_mods="" move_mods=""

trim() { local s="$1"; s="${s#"${s%%[![:space:]]*}"}"; s="${s%"${s##*[![:space:]]}"}"; echo "$s"; }

fmt_mods() {
  local m="$1"
  m="${m//\$mod/S}"; m="${m//SUPER/S}"; m="${m//SHIFT/⇧}"
  m="${m//CTRL/C}"; m="${m//ALT/A}"
  m=$(echo "$m" | sed 's/  */ /g; s/^ //; s/ $//' | tr ' ' '+')
  echo "$m"
}

fmt_exec() {
  local a="$1"
  case "$a" in
    *wezterm*)       echo "终端" ;;
    *rofi*drun*)     echo "启动器" ;;
    *rofi*window*)   echo "窗口列表" ;;
    *emacsclient*monitors*) echo "编辑显示器" ;;
    *emacsclient*)   echo "Emacs" ;;
    *hyprlock*)      echo "锁屏" ;;
    *wlogout*)       echo "电源菜单" ;;
    *nm-connection*) echo "网络设置" ;;
    *cliphist*)      echo "剪贴板" ;;
    *grimblast*area*)    echo "截图(区域)" ;;
    *grimblast*active*)  echo "截图(窗口)" ;;
    *grimblast*screen*)  echo "截图(全屏)" ;;
    *swappy*)        echo "截图+标注" ;;
    *record-toggle*) echo "录屏" ;;
    *keybind-cheatsheet*) echo "快捷键" ;;
    *dolphin*)       echo "文件管理" ;;
    *hyprctl*reload*) echo "重载配置" ;;
    *suspend*)       echo "休眠" ;;
    *) echo "$a" | sed 's|^~/.config/hypr/scripts/||; s| .*||; s|.*/||' ;;
  esac
}

add() {
  local k="$1" v="$2"
  if [[ -z "${raw[$k]+x}" ]]; then
    order+=("$k")
  fi
  raw["$k"]="$v"
}

while IFS= read -r line; do
  line="$(trim "$line")"
  [[ -z "$line" || "$line" =~ ^# || "$line" =~ ^\$ ]] && continue

  if [[ "$line" =~ ^submap\ *=\ *(.+) ]]; then
    sub="$(trim "${BASH_REMATCH[1]}")"
    [[ "$sub" == "reset" ]] && in_submap="" || in_submap="$sub"
    continue
  fi

  [[ "$line" =~ ^bind[eld]*\ *=\ *(.+) ]] || continue
  IFS=',' read -r mods key dispatcher args <<< "${BASH_REMATCH[1]}"
  mods="$(trim "$mods")"; key="$(trim "$key")"
  dispatcher="$(trim "$dispatcher")"; args="$(trim "$args")"
  [[ "$key" == "catchall" ]] && continue

  local_mods="$(fmt_mods "$mods")"
  prefix=""
  [[ -n "$in_submap" ]] && prefix="[X] "

  if [[ -z "$in_submap" && "$dispatcher" == "workspace" && "$key" =~ ^[0-9]$ ]]; then
    ws_mods="$local_mods"
    continue
  fi
  if [[ -z "$in_submap" && "$dispatcher" == "movetoworkspace" && "$key" =~ ^[0-9]$ ]]; then
    move_ws_mods="$local_mods"
    continue
  fi
  if [[ -z "$in_submap" && "$dispatcher" == "movefocus" ]]; then
    nav_mods="$local_mods"
    continue
  fi
  if [[ -z "$in_submap" && "$dispatcher" == "movewindow" ]]; then
    move_mods="$local_mods"
    continue
  fi

  if [[ -n "$local_mods" ]]; then
    combo="${prefix}${local_mods}+${key}"
  else
    combo="${prefix}${key}"
  fi

  case "$dispatcher" in
    exec)                   act="$(fmt_exec "$args")" ;;
    killactive)             act="关闭窗口" ;;
    exit)                   act="退出" ;;
    togglefloating)         act="浮动" ;;
    togglesplit)            act="切换分割" ;;
    cyclenext)              [[ "$args" == *prev* ]] && act="上一窗口" || act="下一窗口" ;;
    workspace)
      case "$args" in
        e-1) act="上一工作区" ;; e+1) act="下一工作区" ;; *) act="工作区$args" ;;
      esac ;;
    movetoworkspace)
      case "$args" in
        e-1) act="移到上一区" ;; e+1) act="移到下一区" ;;
        special:*) act="移到特殊区" ;; *) act="移到工作区$args" ;;
      esac ;;
    togglespecialworkspace) act="特殊工作区" ;;
    submap)
      [[ "$args" == "reset" ]] && act="退出子图" || act="前缀键" ;;
    *) act="$dispatcher $args" ;;
  esac

  add "$combo" "$act"
done < "$CONF"

[[ -n "$nav_mods" ]]     && add "${nav_mods}+B/F/P/N" "焦点 ←→↑↓"
[[ -n "$move_mods" ]]    && add "${move_mods}+B/F/P/N" "移窗 ←→↑↓"
[[ -n "$ws_mods" ]]      && add "${ws_mods}+1~9" "切换工作区"
[[ -n "$move_ws_mods" ]] && add "${move_ws_mods}+1~9" "移到工作区"

lines=()
for k in "${order[@]}"; do
  lines+=("${k}  ${raw[$k]}")
done

n=${#lines[@]}
half=$(( (n + 1) / 2 ))

output=""
for ((i=0; i<half; i++)); do
  left="${lines[$i]}"
  j=$((i + half))
  right=""
  (( j < n )) && right="${lines[$j]}"
  output+="$(printf '%-28s│ %s' "$left" "$right")"$'\n'
done

echo -n "$output" | rofi -dmenu -i -p "⌨" -no-custom \
  -theme-str 'window {width: 52%;} listview {lines: '"$half"'; spacing: 0px; fixed-height: false;} element {padding: 2px 6px;} element-text {font: "CaskaydiaCove Nerd Font Mono 10";}'
