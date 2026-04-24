package ui

func helpFor(t tab, detail bool) string {
	switch t {
	case tabGroups:
		if detail {
			return "↑/↓ move  enter select  t test-group  d test-node  esc back  q quit"
		}
		return "↑/↓ move  enter detail  r refresh  1-6 tabs  q quit"
	case tabProxies:
		return "↑/↓ move  d test  r refresh  1-6 tabs  q quit"
	case tabConns:
		return "↑/↓ scroll  g top  1-6 tabs  q quit"
	case tabLogs:
		return "↑/↓ scroll  a autoscroll  c clear  g bottom  1-6 tabs  q quit"
	case tabTraffic:
		return "1-6 tabs  q quit"
	case tabConfig:
		return "m mode  t tun  r restart  R reload  1-6 tabs  q quit"
	}
	return "1-6 tabs  q quit"
}
