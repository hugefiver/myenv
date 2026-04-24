package ui

func helpFor(t tab, mode string) string {
	switch t {
	case tabGroups:
		if mode == "search" {
			return "type to filter  enter apply  esc cancel"
		}
		return "tab/h/l switch pane  ↑↓ move  enter pick  t test  / search  g GLOBAL  r refresh  q quit"
	case tabProxies:
		return "↑/↓ move  d test  r refresh  1-6 tabs  q quit"
	case tabConns:
		return "↑/↓ move  G bottom  g top  1-6 tabs  q quit"
	case tabLogs:
		return "↑/↓ scroll  a autoscroll  c clear  g bottom  1-6 tabs  q quit"
	case tabTraffic:
		return "1-6 tabs  q quit"
	case tabConfig:
		return "m mode  t tun  r restart  R reload  1-6 tabs  q quit"
	}
	return "1-6 tabs  q quit"
}
