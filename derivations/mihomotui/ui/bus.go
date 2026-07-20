package ui

import (
	"context"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

type requestID uint64

type requestTracker struct {
	next    requestID
	current map[string]requestID
}

func (t *requestTracker) Begin(target string) requestID {
	t.next++
	if t.current == nil {
		t.current = make(map[string]requestID)
	}
	t.current[target] = t.next
	return t.next
}

func (t *requestTracker) IsCurrent(id requestID, target string) bool {
	return t.current != nil && t.current[target] == id
}

type streamClass uint8

const (
	streamConnections streamClass = iota + 1
	streamLogs
	streamTraffic
)

type busMsg struct {
	Inner    tea.Msg
	LogDrops uint64
}

type messageBus struct {
	control     chan tea.Msg
	connections chan tea.Msg
	logs        chan tea.Msg
	traffic     chan tea.Msg

	mu      sync.Mutex
	current map[streamClass]requestID
	dropped atomic.Uint64
}

func newMessageBus() *messageBus {
	return &messageBus{
		control:     make(chan tea.Msg, 32),
		connections: make(chan tea.Msg, 1),
		logs:        make(chan tea.Msg, 256),
		traffic:     make(chan tea.Msg, 1),
		current: map[streamClass]requestID{
			streamConnections: 0,
			streamLogs:        0,
			streamTraffic:     0,
		},
	}
}

func (b *messageBus) queue(class streamClass) chan tea.Msg {
	switch class {
	case streamConnections:
		return b.connections
	case streamLogs:
		return b.logs
	case streamTraffic:
		return b.traffic
	default:
		return nil
	}
}

func (b *messageBus) AdvanceStream(class streamClass, id requestID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current[class] = id
	queue := b.queue(class)
	if queue == nil {
		return
	}
	for {
		select {
		case <-queue:
		default:
			return
		}
	}
}

func (b *messageBus) SendControl(ctx context.Context, msg tea.Msg) bool {
	select {
	case b.control <- msg:
		return true
	case <-ctx.Done():
		return false
	}
}

func (b *messageBus) SendLatest(class streamClass, id requestID, msg tea.Msg) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.current[class] != id {
		return
	}
	queue := b.queue(class)
	if queue == nil {
		return
	}
	select {
	case queue <- msg:
		return
	default:
	}
	select {
	case <-queue:
	default:
	}
	queue <- msg
}

func (b *messageBus) SendLog(id requestID, msg tea.Msg) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.current[streamLogs] != id {
		return
	}
	select {
	case b.logs <- msg:
	default:
		b.dropped.Add(1)
	}
}

func (b *messageBus) Wait(ctx context.Context) tea.Msg {
	if ctx.Err() != nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return nil
	case msg := <-b.control:
		if ctx.Err() != nil {
			return nil
		}
		return busMsg{Inner: msg, LogDrops: b.dropped.Swap(0)}
	case msg := <-b.connections:
		if ctx.Err() != nil {
			return nil
		}
		return busMsg{Inner: msg, LogDrops: b.dropped.Swap(0)}
	case msg := <-b.logs:
		if ctx.Err() != nil {
			return nil
		}
		return busMsg{Inner: msg, LogDrops: b.dropped.Swap(0)}
	case msg := <-b.traffic:
		if ctx.Err() != nil {
			return nil
		}
		return busMsg{Inner: msg, LogDrops: b.dropped.Swap(0)}
	}
}
