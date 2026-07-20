package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const MaxStreamLineBytes = 16 << 20

func (c *Client) streamJSON(ctx context.Context, endpoint string, decode func([]byte) error) error {
	label := requestLabel(http.MethodGet, endpoint)
	req, err := c.newReq(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("%s: request: %w", label, err)
	}
	streamHTTP := *c.HTTP
	streamHTTP.Timeout = 0
	resp, err := streamHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s: request: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return c.statusError(label, resp)
	}

	reader := bufio.NewReaderSize(resp.Body, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		line, readErr := readStreamLine(reader)
		if len(line) > 0 {
			if err := decode(line); err != nil {
				return fmt.Errorf("%s: %w", label, err)
			}
		}
		if readErr != nil {
			return fmt.Errorf("%s: %w", label, readErr)
		}
	}
}

func readStreamLine(reader *bufio.Reader) ([]byte, error) {
	line := make([]byte, 0, 64*1024)
	for {
		fragment, err := reader.ReadSlice('\n')
		bytesBeforeNewline := len(fragment)
		if err == nil && bytesBeforeNewline > 0 {
			bytesBeforeNewline--
		}
		if len(line)+bytesBeforeNewline > MaxStreamLineBytes {
			return nil, fmt.Errorf("stream line exceeds %d bytes", MaxStreamLineBytes)
		}
		line = append(line, fragment...)
		if !errors.Is(err, bufio.ErrBufferFull) {
			return line, err
		}
	}
}

func (c *Client) StreamConnections(ctx context.Context, out chan<- ConnectionsSnapshot) error {
	return c.streamJSON(ctx, "/connections", func(data []byte) error {
		var snapshot ConnectionsSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		select {
		case out <- snapshot:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (c *Client) StreamLogs(ctx context.Context, level string, out chan<- LogEntry) error {
	return c.streamJSON(ctx, "/logs?level="+level, func(data []byte) error {
		var entry LogEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		select {
		case out <- entry:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (c *Client) StreamTraffic(ctx context.Context, out chan<- Traffic) error {
	return c.streamJSON(ctx, "/traffic", func(data []byte) error {
		var traffic Traffic
		if err := json.Unmarshal(data, &traffic); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		select {
		case out <- traffic:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}
