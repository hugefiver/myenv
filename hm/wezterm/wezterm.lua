local wezterm = require("wezterm")
local act = wezterm.action

return {
  font = wezterm.font_with_fallback({
    "CaskaydiaCove Nerd Font Mono",
    "Noto Color Emoji",
  }),
  font_size = 12.5,
  color_scheme = "Kanagawa (Gogh)",
  default_prog = { "zsh", "-l" },
  adjust_window_size_when_changing_font_size = false,
  window_background_opacity = 0.65,
  enable_tab_bar = true,
  hide_tab_bar_if_only_one_tab = true,
  use_fancy_tab_bar = false,
  default_cursor_style = "BlinkingBar",
  window_padding = {
    left = 10,
    right = 10,
    top = 8,
    bottom = 8,
  },
  keys = {
    { key = "Enter", mods = "ALT", action = act.ToggleFullScreen },
  },
  mouse_bindings = {
    {
      event = { Up = { streak = 1, button = "Right" } },
      mods = "NONE",
      action = act.PasteFrom("Clipboard"),
    },
    -- 取消「单击链接直接打开浏览器」的默认行为：
    -- 左键单击释放只做选区/取消选区，不再跟随 link。
    {
      event = { Up = { streak = 1, button = "Left" } },
      mods = "NONE",
      action = act.CompleteSelection("ClipboardAndPrimarySelection"),
    },
    -- Ctrl + 左键单击：打开光标下的链接。
    {
      event = { Up = { streak = 1, button = "Left" } },
      mods = "CTRL",
      action = act.OpenLinkAtMouseCursor,
    },
    -- 阻止 Ctrl+左键 在按下时移动光标 / 触发其它默认动作。
    {
      event = { Down = { streak = 1, button = "Left" } },
      mods = "CTRL",
      action = wezterm.action.Nop,
    },
  },
}
