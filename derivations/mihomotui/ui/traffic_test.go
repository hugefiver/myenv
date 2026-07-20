package ui

import (
	"context"
	"strings"
	"testing"

	"mihomotui/api"
)

func TestTrafficUsesRatesAndReportedTotals(t *testing.T) {
	model := newTrafficModel(context.Background(), nil, newMessageBus())
	model.width, model.height = 100, 20
	sample := api.Traffic{Up: 123, Down: 456, UpTotal: 5 << 20, DownTotal: 7 << 20}

	model.Update(trafficMsg{t: sample})

	if model.cur.Up != sample.Up || model.cur.Down != sample.Down {
		t.Fatalf("current rates = up %d down %d, want up %d down %d", model.cur.Up, model.cur.Down, sample.Up, sample.Down)
	}
	if model.totalUp != sample.UpTotal || model.totalDn != sample.DownTotal {
		t.Fatalf("session totals = up %d down %d, want up %d down %d", model.totalUp, model.totalDn, sample.UpTotal, sample.DownTotal)
	}
	view := model.View()
	if !strings.Contains(view, "Upload   "+humanBytes(sample.Up)+"/s") || !strings.Contains(view, "session ↑"+humanBytes(sample.UpTotal)+"  ↓"+humanBytes(sample.DownTotal)) {
		t.Fatalf("View() does not distinguish rates from totals: %q", view)
	}
}
