local wezterm = require("wezterm")
local act = wezterm.action

return {
  font = wezterm.font_with_fallback({
    "CaskaydiaCove Nerd Font Mono",
    "Noto Color Emoji",
  }),
  font_size = 12.5,
  color_scheme = "Catppuccin Mocha",
  default_prog = { "zsh", "-l" },
  adjust_window_size_when_changing_font_size = false,
  window_background_opacity = 0.94,
  enable_tab_bar = true,
  hide_tab_bar_if_only_one_tab = true,
  use_fancy_tab_bar = false,
  default_cursor_style = "BlinkingBar",
  copy_on_select = true,
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
      action = act.InputSelector {
        title = "Menu",
        choices = {
          { label = "Copy",  id = "copy" },
          { label = "Paste", id = "paste" },
        },
        action = wezterm.action_callback(function(window, pane, id)
          if id == "copy" then
            window:perform_action(act.CopyTo("Clipboard"), pane)
          elseif id == "paste" then
            window:perform_action(act.PasteFrom("Clipboard"), pane)
          end
        end),
      },
    },
  },
}