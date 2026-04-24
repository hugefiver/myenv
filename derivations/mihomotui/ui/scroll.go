package ui

func clampScroll(cursor, scroll, viewHeight, totalRows int) int {
	if viewHeight <= 0 || totalRows <= 0 {
		return 0
	}
	margin := 1
	if viewHeight < 4 {
		margin = 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= totalRows {
		cursor = totalRows - 1
	}
	if cursor-scroll < margin {
		scroll = cursor - margin
	}
	if cursor-scroll > viewHeight-1-margin {
		scroll = cursor - (viewHeight - 1 - margin)
	}
	maxScroll := totalRows - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}
