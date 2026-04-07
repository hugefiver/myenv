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
  },
}
