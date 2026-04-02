#!/usr/bin/env bash
set -euo pipefail

CONF="${XDG_CONFIG_HOME:-$HOME/.config}/hypr/keybinds.conf"
[[ -f "$CONF" ]] || { notify-send "Keybind Cheatsheet" "keybinds.conf not found"; exit 1; }

section=""
entries=()

fmt_mods() {
  local m="$1"
  m="${m//\$mod/Super}"; m="${m//SUPER/Super}"; m="${m//SHIFT/Shift}"
  m="${m//CTRL/Ctrl}"; m="${m//ALT/Alt}"
  m=$(echo "$m" | sed 's/  */ /g; s/^ //; s/ $//' | tr ' ' '+')
  echo "$m"
}

fmt_action() {
  local dispatcher="$1" args="$2"
  case "$dispatcher" in
    exec)
      local cmd
      cmd=$(echo "$args" | sed 's|^~/.config/hypr/scripts/||; s| .*||; s|.*/||')
      echo "$cmd" ;;
    killactive)     echo "关闭窗口" ;;
    exit)           echo "退出 Hyprland" ;;
    togglefloating) echo "切换浮动" ;;
    togglesplit)    echo "切换分割方向" ;;
    cyclenext)
      [[ "$args" == *prev* ]] && echo "上一个窗口" || echo "下一个窗口" ;;
    movefocus)
      case "$args" in
        l) echo "聚焦 ← 左" ;; r) echo "聚焦 → 右" ;;
        u) echo "聚焦 ↑ 上" ;; d) echo "聚焦 ↓ 下" ;; *) echo "聚焦 $args" ;;
      esac ;;
    movewindow)
      case "$args" in
        l) echo "移动窗口 ← 左" ;; r) echo "移动窗口 → 右" ;;
        u) echo "移动窗口 ↑ 上" ;; d) echo "移动窗口 ↓ 下" ;; *) echo "移动窗口 $args" ;;
      esac ;;
    workspace)
      case "$args" in
        e-1) echo "上一个工作区" ;; e+1) echo "下一个工作区" ;;
        *)   echo "工作区 $args" ;;
      esac ;;
    movetoworkspace)
      case "$args" in
        e-1) echo "移到上一个工作区" ;; e+1) echo "移到下一个工作区" ;;
        special:*) echo "移到特殊工作区" ;;
        *)   echo "移到工作区 $args" ;;
      esac ;;
    togglespecialworkspace) echo "切换特殊工作区" ;;
    submap)
      [[ "$args" == "reset" ]] && echo "退出子图" || echo "进入 [$args] 子图" ;;
    *) echo "$dispatcher $args" ;;
  esac
}

in_submap=""

while IFS= read -r line; do
  line="${line#"${line%%[![:space:]]*}"}"
  [[ -z "$line" ]] && continue

  # 注释行 → 分区标题（去掉装饰用的 ─━═ 线条）
  if [[ "$line" =~ ^#\ *(.+) ]]; then
    header="${BASH_REMATCH[1]}"
    header=$(echo "$header" | sed 's/[─━═]//g; s/^[[:space:]]*//; s/[[:space:]]*$//')
    [[ -n "$header" ]] && section="$header"
    continue
  fi

  # submap 状态机：进入/退出子图影响后续 bind 的前缀显示
  if [[ "$line" =~ ^submap\ *=\ *(.+) ]]; then
    sub="${BASH_REMATCH[1]}"
    sub="${sub#"${sub%%[![:space:]]*}"}"
    [[ "$sub" == "reset" ]] && in_submap="" || in_submap="$sub"
    continue
  fi

  [[ "$line" =~ ^\$ ]] && continue

  if [[ "$line" =~ ^bind[eld]*\ *=\ *(.+) ]]; then
    IFS=',' read -r mods key dispatcher args <<< "${BASH_REMATCH[1]}"
    mods="${mods#"${mods%%[![:space:]]*}"}"; mods="${mods%"${mods##*[![:space:]]}"}"
    key="${key#"${key%%[![:space:]]*}"}"; key="${key%"${key##*[![:space:]]}"}"
    dispatcher="${dispatcher#"${dispatcher%%[![:space:]]*}"}"; dispatcher="${dispatcher%"${dispatcher##*[![:space:]]}"}"
    args="${args#"${args%%[![:space:]]*}"}"; args="${args%"${args##*[![:space:]]}"}"

    [[ "$key" == "catchall" ]] && continue

    if [[ -n "$mods" ]]; then
      combo="$(fmt_mods "$mods")+${key}"
    else
      combo="$key"
    fi
    [[ -n "$in_submap" ]] && combo="[${in_submap}] ${combo}"

    action="$(fmt_action "$dispatcher" "$args")"
    printf -v entry "%-30s  %s" "$combo" "$action"

    if [[ -n "$section" ]]; then
      entries+=("── $section ──")
      section=""
    fi
    entries+=("$entry")
  fi
done < "$CONF"

printf '%s\n' "${entries[@]}" | rofi -dmenu -i -p "⌨ 快捷键" -no-custom -theme-str 'window {width: 50%;} listview {lines: 30;}'
