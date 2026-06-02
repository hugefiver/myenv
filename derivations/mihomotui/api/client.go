package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const TestURL = "https://www.gstatic.com/generate_204"

type Client struct {
	Base   string
	Secret string
	HTTP   *http.Client
}

func New(base, secret string) *Client {
	base = strings.TrimRight(base, "/")
	return &Client{
		Base:   base,
		Secret: secret,
		HTTP:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) newReq(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rdr)
	if err != nil {
		return nil, err
	}
	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	req, err := c.newReq(ctx, method, path, body)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Proxies(ctx context.Context) (map[string]Proxy, error) {
	var r ProxiesResp
	if err := c.do(ctx, "GET", "/proxies", nil, &r); err != nil {
		return nil, err
	}
	return r.Proxies, nil
}

func (c *Client) SelectProxy(ctx context.Context, group, node string) error {
	return c.do(ctx, "PUT", "/proxies/"+url.PathEscape(group), map[string]string{"name": node}, nil)
}

func (c *Client) ProxyDelay(ctx context.Context, name string, timeoutMs int) (int, error) {
	q := fmt.Sprintf("?url=%s&timeout=%d", url.QueryEscape(TestURL), timeoutMs)
	var r DelayResp
	if err := c.do(ctx, "GET", "/proxies/"+url.PathEscape(name)+"/delay"+q, nil, &r); err != nil {
		return 0, err
	}
	if r.Message != "" {
		return 0, fmt.Errorf("%s", r.Message)
	}
	return r.Delay, nil
}

func (c *Client) GroupDelay(ctx context.Context, name string, timeoutMs int) (GroupDelayResp, error) {
	q := fmt.Sprintf("?url=%s&timeout=%d", url.QueryEscape(TestURL), timeoutMs)
	var r GroupDelayResp
	if err := c.do(ctx, "GET", "/group/"+url.PathEscape(name)+"/delay"+q, nil, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) Configs(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := c.do(ctx, "GET", "/configs", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Client) PatchConfig(ctx context.Context, body any) error {
	return c.do(ctx, "PATCH", "/configs", body, nil)
}

func (c *Client) SetMode(ctx context.Context, mode string) error {
	return c.PatchConfig(ctx, map[string]string{"mode": mode})
}

func (c *Client) SetTun(ctx context.Context, enable bool) error {
	return c.PatchConfig(ctx, map[string]any{"tun": map[string]any{"enable": enable}})
}

func (c *Client) PutConfig(ctx context.Context, path string) error {
	return c.do(ctx, "PUT", "/configs?force=true", map[string]string{"path": path}, nil)
}

func (c *Client) DownloadConfig(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("bad url: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	f, err := os.CreateTemp("", "mihomo-profile-*.yaml")
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write failed: %w", err)
	}
	return f.Name(), nil
}
