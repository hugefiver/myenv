package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func streamClient(t *testing.T, body io.Reader, status int) *Client {
	t.Helper()
	c := newClientForURL(t, "https://controller.test")
	c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(body), Request: req}, nil
	})}
	return c
}

func TestStreamJSONUsesInjectedClientAndReturnsEndpointEOF(t *testing.T) {
	var calls atomic.Int32
	c := newClientForURL(t, "https://controller.test")
	c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"up":1,"down":2,"upTotal":3,"downTotal":4}` + "\n")), Request: req}, nil
	})}
	out := make(chan Traffic, 1)
	err := c.StreamTraffic(context.Background(), out)
	if err == nil || !strings.Contains(err.Error(), "GET /traffic") || !errors.Is(err, io.EOF) {
		t.Fatalf("StreamTraffic() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("RoundTripper calls = %d, want 1", calls.Load())
	}
	got := <-out
	if got.Up != 1 || got.Down != 2 || got.UpTotal != 3 || got.DownTotal != 4 {
		t.Fatalf("Traffic = %#v", got)
	}
}

func TestStreamJSONEnforcesExactLineLimit(t *testing.T) {
	const prefix = `{"payload":"`
	const suffix = `"}`
	exact := prefix + strings.Repeat("x", MaxStreamLineBytes-len(prefix)-len(suffix)) + suffix + "\n"
	c := streamClient(t, strings.NewReader(exact), http.StatusOK)
	decoded := false
	err := c.streamJSON(context.Background(), "/traffic", func(line []byte) error {
		decoded = len(line) == MaxStreamLineBytes+1
		return nil
	})
	if err == nil || !errors.Is(err, io.EOF) || !decoded {
		t.Fatalf("streamJSON() error = %v, decoded exact line = %v", err, decoded)
	}

	over := strings.Repeat("x", MaxStreamLineBytes+1) + "\n"
	c = streamClient(t, strings.NewReader(over), http.StatusOK)
	err = c.streamJSON(context.Background(), "/traffic", func([]byte) error {
		t.Fatal("decoder called for oversized line")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "stream line exceeds 16777216 bytes") {
		t.Fatalf("streamJSON() error = %v", err)
	}
}

func TestStreamsReturnDecodeErrorsWithoutSendingPseudoMessages(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Client) (<-chan struct{}, error)
	}{
		{
			name: "connections",
			run: func(c *Client) (<-chan struct{}, error) {
				out := make(chan ConnectionsSnapshot, 1)
				err := c.StreamConnections(context.Background(), out)
				done := make(chan struct{})
				if len(out) != 0 {
					t.Fatal("StreamConnections() sent a pseudo message")
				}
				close(done)
				return done, err
			},
		},
		{
			name: "logs",
			run: func(c *Client) (<-chan struct{}, error) {
				out := make(chan LogEntry, 1)
				err := c.StreamLogs(context.Background(), "info", out)
				done := make(chan struct{})
				if len(out) != 0 {
					t.Fatal("StreamLogs() sent a pseudo message")
				}
				close(done)
				return done, err
			},
		},
		{
			name: "traffic",
			run: func(c *Client) (<-chan struct{}, error) {
				out := make(chan Traffic, 1)
				err := c.StreamTraffic(context.Background(), out)
				done := make(chan struct{})
				if len(out) != 0 {
					t.Fatal("StreamTraffic() sent a pseudo message")
				}
				close(done)
				return done, err
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.run(streamClient(t, strings.NewReader("not JSON\n"), http.StatusOK))
			if err == nil || !strings.Contains(err.Error(), "decode") || !strings.Contains(err.Error(), "GET /") {
				t.Fatalf("stream error = %v", err)
			}
		})
	}
}

func TestStreamJSONReportsNonSuccessAndCancellation(t *testing.T) {
	c := streamClient(t, strings.NewReader("\x1b[31mdenied\x1b[0m"), http.StatusServiceUnavailable)
	err := c.streamJSON(context.Background(), "/traffic", func([]byte) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "GET /traffic") || !strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("streamJSON() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = streamClient(t, strings.NewReader(""), http.StatusOK).streamJSON(ctx, "/traffic", func([]byte) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("streamJSON() error = %v, want context.Canceled", err)
	}
}

type gatedReader struct {
	started chan<- struct{}
	release <-chan struct{}
	ctx     context.Context
	data    []byte
	emitted bool
}

func (r *gatedReader) Read(p []byte) (int, error) {
	if r.emitted {
		return 0, io.EOF
	}
	r.emitted = true
	close(r.started)
	select {
	case <-r.release:
		return copy(p, r.data), nil
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	}
}

func TestStreamTrafficIgnoresWholeResponseClientTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	c := newClientForURL(t, "https://controller.test")
	c.HTTP = &http.Client{
		Timeout: 5 * time.Millisecond,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body := &gatedReader{
				started: started,
				release: release,
				ctx:     req.Context(),
				data:    []byte(`{"up":1,"down":2}` + "\n"),
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Request: req}, nil
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out := make(chan Traffic, 1)
	result := make(chan error, 1)
	go func() {
		result <- c.StreamTraffic(ctx, out)
	}()

	select {
	case <-started:
	case <-ctx.Done():
		t.Fatalf("stream did not start: %v", ctx.Err())
	}

	timer := time.NewTimer(50 * time.Millisecond)
	select {
	case <-timer.C:
		close(release)
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		t.Fatalf("stream timed out before release: %v", ctx.Err())
	}

	select {
	case err := <-result:
		if err == nil || !errors.Is(err, io.EOF) || !strings.Contains(err.Error(), "GET /traffic") {
			t.Fatalf("StreamTraffic() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("stream did not finish: %v", ctx.Err())
	}

	if got := <-out; got.Up != 1 || got.Down != 2 {
		t.Fatalf("Traffic = %#v", got)
	}
}
