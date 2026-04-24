package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
)

func (c *Client) streamJSON(ctx context.Context, path string, decode func([]byte) error) error {
	req, err := c.newReq(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rd := bufio.NewReaderSize(resp.Body, 64*1024)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line, err := rd.ReadBytes('\n')
		if len(line) > 0 {
			if dErr := decode(line); dErr != nil {
				return dErr
			}
		}
		if err != nil {
			return err
		}
	}
}

func (c *Client) StreamConnections(ctx context.Context, out chan<- ConnectionsSnapshot) error {
	return c.streamJSON(ctx, "/connections", func(b []byte) error {
		var s ConnectionsSnapshot
		if err := json.Unmarshal(b, &s); err != nil {
			return nil
		}
		select {
		case out <- s:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
}

func (c *Client) StreamLogs(ctx context.Context, level string, out chan<- LogEntry) error {
	path := "/logs?level=" + level
	return c.streamJSON(ctx, path, func(b []byte) error {
		var e LogEntry
		if err := json.Unmarshal(b, &e); err != nil {
			return nil
		}
		select {
		case out <- e:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
}

func (c *Client) StreamTraffic(ctx context.Context, out chan<- Traffic) error {
	return c.streamJSON(ctx, "/traffic", func(b []byte) error {
		var t Traffic
		if err := json.Unmarshal(b, &t); err != nil {
			return nil
		}
		select {
		case out <- t:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
}
