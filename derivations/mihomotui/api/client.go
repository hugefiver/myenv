package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mihomotui/internal/termtext"
)

const (
	TestURL             = "https://www.gstatic.com/generate_204"
	MaxBodyBytes  int64 = 32 << 20
	MaxErrorBytes int64 = 64 << 10
)

type Client struct {
	baseURL *url.URL
	Secret  string
	HTTP    *http.Client
}

func New(base, secret string) (*Client, error) {
	baseURL, err := parseBaseURL(base)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: baseURL,
		Secret:  secret,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return nil, errors.New("controller URL is empty")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse controller URL: %w", err)
	}
	if !u.IsAbs() || u.Hostname() == "" {
		return nil, errors.New("controller URL must be absolute with a host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("controller URL scheme must be http or https")
	}
	if u.User != nil {
		return nil, errors.New("controller URL must not contain userinfo")
	}
	if u.RawQuery != "" || u.ForceQuery {
		return nil, errors.New("controller URL must not contain a query")
	}
	if u.Fragment != "" {
		return nil, errors.New("controller URL must not contain a fragment")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return u, nil
}

func validateDownloadURL(u *url.URL) error {
	if u == nil || !u.IsAbs() || u.Hostname() == "" {
		return errors.New("download URL must be absolute with a host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("download URL scheme must be http or https")
	}
	if u.User != nil {
		return errors.New("download URL must not contain userinfo")
	}
	return nil
}

func (c *Client) Warning() string {
	if c.baseURL.Scheme == "https" {
		return ""
	}
	host := c.baseURL.Hostname()
	if strings.EqualFold(host, "localhost") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return ""
	}
	return "plaintext HTTP sends the Mihomo secret"
}

func (c *Client) endpointURL(endpoint string) (*url.URL, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	if endpointURL.IsAbs() || endpointURL.Host != "" {
		return nil, errors.New("endpoint must be relative")
	}

	u := *c.baseURL
	basePath := strings.TrimRight(u.EscapedPath(), "/")
	endpointPath := strings.TrimLeft(endpointURL.EscapedPath(), "/")
	rawPath := basePath + "/" + endpointPath
	if rawPath == "" {
		rawPath = "/"
	}
	path, err := url.PathUnescape(rawPath)
	if err != nil {
		return nil, fmt.Errorf("decode endpoint path: %w", err)
	}
	u.Path = path
	u.RawPath = rawPath
	u.RawQuery = endpointURL.RawQuery
	u.ForceQuery = endpointURL.ForceQuery
	u.Fragment = ""
	u.RawFragment = ""
	return &u, nil
}

func (c *Client) newReq(ctx context.Context, method, endpoint string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	u, err := c.endpointURL(endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
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

func requestLabel(method, endpoint string) string {
	return method + " " + endpoint
}

func readBounded(body io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return data, nil
}

func (c *Client) statusError(label string, resp *http.Response) error {
	data, err := readBounded(resp.Body, MaxErrorBytes)
	detail := ""
	if err != nil {
		detail = termtext.SingleLine(err.Error())
	} else {
		detail = strings.TrimSpace(termtext.SingleLine(string(data)))
	}
	if detail == "" {
		return fmt.Errorf("%s: http %d", label, resp.StatusCode)
	}
	return fmt.Errorf("%s: http %d: %s", label, resp.StatusCode, detail)
}

func (c *Client) do(ctx context.Context, method, endpoint string, body any, out any) error {
	label := requestLabel(method, endpoint)
	req, err := c.newReq(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("%s: request: %w", label, err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s: request: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return c.statusError(label, resp)
	}
	data, err := readBounded(resp.Body, MaxBodyBytes)
	if err != nil {
		return fmt.Errorf("%s: read response: %w", label, err)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s: decode: %w", label, err)
	}
	return nil
}

func (c *Client) Proxies(ctx context.Context) (map[string]Proxy, error) {
	var response ProxiesResp
	if err := c.do(ctx, http.MethodGet, "/proxies", nil, &response); err != nil {
		return nil, err
	}
	return response.Proxies, nil
}

func (c *Client) Groups(ctx context.Context) ([]Proxy, error) {
	var response GroupsResp
	if err := c.do(ctx, http.MethodGet, "/group", nil, &response); err != nil {
		return nil, err
	}
	return response.Proxies, nil
}

func (c *Client) SelectProxy(ctx context.Context, group, node string) error {
	return c.do(ctx, http.MethodPut, "/proxies/"+url.PathEscape(group), map[string]string{"name": node}, nil)
}

func (c *Client) ProxyDelay(ctx context.Context, name string, timeoutMs int) (int, error) {
	query := fmt.Sprintf("?url=%s&timeout=%d", url.QueryEscape(TestURL), timeoutMs)
	endpoint := "/proxies/" + url.PathEscape(name) + "/delay" + query
	var response DelayResp
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return 0, err
	}
	if response.Message != "" {
		return 0, fmt.Errorf("GET %s: %s", endpoint, termtext.SingleLine(response.Message))
	}
	return response.Delay, nil
}

func (c *Client) GroupDelay(ctx context.Context, name string, timeoutMs int) (GroupDelayResp, error) {
	query := fmt.Sprintf("?url=%s&timeout=%d", url.QueryEscape(TestURL), timeoutMs)
	var response GroupDelayResp
	if err := c.do(ctx, http.MethodGet, "/group/"+url.PathEscape(name)+"/delay"+query, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) Configs(ctx context.Context) (*Config, error) {
	var config Config
	if err := c.do(ctx, http.MethodGet, "/configs", nil, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Client) PatchConfig(ctx context.Context, body any) error {
	return c.do(ctx, http.MethodPatch, "/configs", body, nil)
}

func (c *Client) SetMode(ctx context.Context, mode string) error {
	return c.PatchConfig(ctx, map[string]string{"mode": mode})
}

func (c *Client) SetTun(ctx context.Context, enable bool) error {
	return c.PatchConfig(ctx, map[string]any{"tun": map[string]any{"enable": enable}})
}

func (c *Client) PutConfig(ctx context.Context, payload []byte) error {
	return c.do(ctx, http.MethodPut, "/configs?force=true", map[string]string{"payload": string(payload)}, nil)
}

func (c *Client) DownloadProfile(ctx context.Context, rawURL string) ([]byte, string, error) {
	downloadURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", errors.New("GET /: invalid download URL")
	}
	if err := validateDownloadURL(downloadURL); err != nil {
		return nil, "", fmt.Errorf("GET %s: %s", downloadEndpoint(downloadURL), termtext.SingleLine(err.Error()))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("GET %s: request: %w", downloadEndpoint(downloadURL), err)
	}
	req.Header.Del("Authorization")

	downloadHTTP := *c.HTTP
	// Profile operations provide their own deadline, which may intentionally be
	// longer than the controller client's general request timeout.
	downloadHTTP.Timeout = 0
	originalCheckRedirect := downloadHTTP.CheckRedirect
	downloadHTTP.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateDownloadURL(req.URL); err != nil {
			return fmt.Errorf("unsafe profile redirect: %w", err)
		}
		if originalCheckRedirect != nil {
			if err := originalCheckRedirect(req, via); err != nil {
				return err
			}
		} else if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if err := validateDownloadURL(req.URL); err != nil {
			return fmt.Errorf("unsafe profile redirect: %w", err)
		}
		req.Header.Del("Authorization")
		return nil
	}

	label := requestLabel(http.MethodGet, downloadEndpoint(downloadURL))
	resp, err := downloadHTTP.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%s: download: %w", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", c.statusError(label, resp)
	}
	data, err := readBounded(resp.Body, MaxBodyBytes)
	if err != nil {
		return nil, "", fmt.Errorf("%s: read response: %w", label, err)
	}
	finalURL := downloadURL.String()
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return data, finalURL, nil
}

func downloadEndpoint(u *url.URL) string {
	endpoint := u.EscapedPath()
	if endpoint == "" {
		endpoint = "/"
	}
	if u.RawQuery != "" {
		endpoint += "?" + u.RawQuery
	}
	return endpoint
}
