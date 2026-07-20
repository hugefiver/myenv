package ui

func helpFor(t tab, mode string) string {
	switch t {
	case tabGroups:
		if mode == "search" {
			return "type to filter  enter apply  esc cancel"
		}
		return "tab/h/l switch pane  ↑↓ move  enter pick  t test  T auto-test  / search  g GLOBAL  r refresh  q quit"
	case tabProxies:
		return "↑/↓ move  d test  r refresh  1-7 tabs  q quit"
	case tabConns:
		return "↑/↓ move  G bottom  g top  r retry  1-7 tabs  q quit"
	case tabLogs:
		return "↑/↓ scroll  a autoscroll  c clear  g bottom  r retry  1-7 tabs  q quit"
	case tabTraffic:
		return "r retry  1-7 tabs  q quit"
	case tabProfiles:
		return "i import  enter apply  d delete  R refresh URL  ↑↓ move  1-7 tabs  q quit"
	case tabConfig:
		return "m mode  t tun  r refresh  1-7 tabs  q quit"
	}
	return "1-7 tabs  q quit"
}
