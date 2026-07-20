package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type shortReader struct {
	data []byte
	done bool
}

func (r *shortReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, r.data), nil
	}
	return 0, io.ErrUnexpectedEOF
}

type repeatReader struct {
	remaining int64
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := int64(len(p))
	if n > r.remaining {
		n = r.remaining
	}
	for i := range p[:n] {
		p[i] = 'x'
	}
	r.remaining -= n
	return int(n), nil
}

func newClientForURL(t *testing.T, rawURL string) *Client {
	t.Helper()
	c, err := New(rawURL, "controller-secret")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

func TestNewValidatesBaseAndWarning(t *testing.T) {
	invalid := []string{
		"",
		"/relative",
		"http://:9090",
		"ftp://example.test",
		"https://user@example.test",
		"https://example.test?query=1",
		"https://example.test#fragment",
	}
	for _, rawURL := range invalid {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := New(rawURL, "secret"); err == nil {
				t.Fatalf("New(%q) succeeded", rawURL)
			}
		})
	}

	for _, tc := range []struct {
		base string
		want string
	}{
		{"https://controller.test", ""},
		{"http://localhost:9090", ""},
		{"http://127.0.0.1:9090", ""},
		{"http://[::1]:9090", ""},
		{"http://controller.test", "plaintext HTTP sends the Mihomo secret"},
	} {
		t.Run(tc.base, func(t *testing.T) {
			c := newClientForURL(t, tc.base)
			if got := c.Warning(); got != tc.want {
				t.Fatalf("Warning() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewRequestPreservesPrefixAndEscapedSegments(t *testing.T) {
	c := newClientForURL(t, "https://controller.test/controller/v1/")
	req, err := c.newReq(context.Background(), http.MethodGet, "/proxies/"+url.PathEscape("A/B"), nil)
	if err != nil {
		t.Fatalf("newReq() error = %v", err)
	}
	if got, want := req.URL.EscapedPath(), "/controller/v1/proxies/A%2FB"; got != want {
		t.Fatalf("EscapedPath() = %q, want %q", got, want)
	}
}

func TestGroupsUsesPrefixAndSanitizesNonSuccessResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/controller/v1/group"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusFound)
		_, _ = io.WriteString(w, "\x1b[31munsafe\x1b[0m\n")
	}))
	defer server.Close()

	c := newClientForURL(t, server.URL+"/controller/v1")
	_, err := c.Groups(context.Background())
	if err == nil {
		t.Fatal("Groups() succeeded")
	}
	if got := err.Error(); !strings.Contains(got, "GET /group") || !strings.Contains(got, "302") || strings.Contains(got, "\x1b") {
		t.Fatalf("Groups() error = %q", got)
	}
}

func TestGroupsDecodesWrappedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/controller/v1/group"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer controller-secret"; got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		_, _ = io.WriteString(w, `{"proxies":[{"name":"Selector","type":"Selector"}]}`)
	}))
	defer server.Close()

	groups, err := newClientForURL(t, server.URL+"/controller/v1").Groups(context.Background())
	if err != nil {
		t.Fatalf("Groups() error = %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Selector" || groups[0].Type != "Selector" {
		t.Fatalf("Groups() = %#v", groups)
	}
}

func TestGroupsRejectsNonSuccessStatuses(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c := newClientForURL(t, "https://controller.test")
			c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader("failure")),
					Request:    req,
				}, nil
			})}
			_, err := c.Groups(context.Background())
			if err == nil || !strings.Contains(err.Error(), "GET /group") || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("Groups() error = %v", err)
			}
		})
	}
}

func TestDoBoundsSuccessfulBodiesByActualBytes(t *testing.T) {
	c := newClientForURL(t, "https://controller.test")
	c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusNoContent,
			ContentLength: 1,
			Body:          io.NopCloser(&repeatReader{remaining: MaxBodyBytes + 1}),
			Request:       req,
		}, nil
	})}

	err := c.do(context.Background(), http.MethodPut, "/configs", map[string]string{"payload": "x"}, nil)
	if err == nil || !strings.Contains(err.Error(), "PUT /configs") || !strings.Contains(err.Error(), "response body exceeds") {
		t.Fatalf("do() error = %v", err)
	}
}

func TestGroupsWrapsShortAndMalformedBodies(t *testing.T) {
	for name, body := range map[string]io.Reader{
		"short":     &shortReader{data: []byte(`{"proxies":`)},
		"malformed": strings.NewReader(`{"proxies":`),
	} {
		t.Run(name, func(t *testing.T) {
			c := newClientForURL(t, "https://controller.test")
			c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Request: req}, nil
			})}
			_, err := c.Groups(context.Background())
			if err == nil || !strings.Contains(err.Error(), "GET /group") {
				t.Fatalf("Groups() error = %v", err)
			}
			if name == "short" && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("Groups() error does not wrap io.ErrUnexpectedEOF: %v", err)
			}
		})
	}
}

func TestPutConfigSendsPayloadAndAcceptsNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPut; got != want {
			t.Errorf("method = %q, want %q", got, want)
		}
		if got, want := r.URL.RequestURI(), "/configs?force=true"; got != want {
			t.Errorf("request URI = %q, want %q", got, want)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(body), `{"payload":"mixed-port: 7890\n"}`; got != want {
			t.Errorf("body = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newClientForURL(t, server.URL)
	if err := c.PutConfig(context.Background(), []byte("mixed-port: 7890\n")); err != nil {
		t.Fatalf("PutConfig() error = %v", err)
	}
}

func TestDownloadProfileDoesNotSendControllerSecret(t *testing.T) {
	var firstAuthorization, targetAuthorization string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetAuthorization = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, "profile")
	}))
	defer target.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstAuthorization = r.Header.Get("Authorization")
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer first.Close()

	c := newClientForURL(t, first.URL)
	c.HTTP.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.Header.Set("Authorization", "Bearer injected-by-policy")
		return nil
	}

	data, finalURL, err := c.DownloadProfile(context.Background(), first.URL)
	if err != nil {
		t.Fatalf("DownloadProfile() error = %v", err)
	}
	if got, want := string(data), "profile"; got != want {
		t.Fatalf("data = %q, want %q", got, want)
	}
	if got, want := finalURL, target.URL; got != want {
		t.Fatalf("final URL = %q, want %q", got, want)
	}
	if firstAuthorization != "" || targetAuthorization != "" {
		t.Fatalf("download Authorization headers = first %q, target %q", firstAuthorization, targetAuthorization)
	}

	var apiAuthorization string
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAuthorization = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"proxies":[]}`)
	}))
	defer controller.Close()
	c = newClientForURL(t, controller.URL)
	if _, err := c.Groups(context.Background()); err != nil {
		t.Fatalf("Groups() error = %v", err)
	}
	if got, want := apiAuthorization, "Bearer controller-secret"; got != want {
		t.Fatalf("normal API Authorization = %q, want %q", got, want)
	}
}

func TestDownloadProfileRejectsUnsafeRedirectAfterPolicy(t *testing.T) {
	var targetHits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetHits.Add(1)
	}))
	defer target.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer first.Close()

	c := newClientForURL(t, first.URL)
	c.HTTP.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.URL.User = url.User("attacker")
		return nil
	}
	_, _, err := c.DownloadProfile(context.Background(), first.URL)
	if err == nil || !strings.Contains(err.Error(), "unsafe profile redirect") || !strings.Contains(err.Error(), "userinfo") {
		t.Fatalf("DownloadProfile() error = %v", err)
	}
	if got := targetHits.Load(); got != 0 {
		t.Fatalf("unsafe redirect reached target %d times", got)
	}
}

func TestDownloadProfileRejectsUnsafeRedirectBeforePolicy(t *testing.T) {
	var targetHits atomic.Int32
	var targetAuthorization string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits.Add(1)
		targetAuthorization = r.Header.Get("Authorization")
	}))
	defer target.Close()
	unsafeTarget := "http://attacker@" + strings.TrimPrefix(target.URL, "http://")
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, unsafeTarget, http.StatusFound)
	}))
	defer first.Close()

	c := newClientForURL(t, first.URL)
	policyCalled := false
	c.HTTP.CheckRedirect = func(*http.Request, []*http.Request) error {
		policyCalled = true
		return nil
	}
	_, _, err := c.DownloadProfile(context.Background(), first.URL)
	if err == nil || !strings.Contains(err.Error(), "unsafe profile redirect") || !strings.Contains(err.Error(), "userinfo") {
		t.Fatalf("DownloadProfile() error = %v", err)
	}
	if policyCalled {
		t.Fatal("custom redirect policy ran for an unsafe target")
	}
	if got := targetHits.Load(); got != 0 {
		t.Fatalf("unsafe redirect reached target %d times", got)
	}
	if targetAuthorization != "" {
		t.Fatalf("unsafe redirect sent Authorization %q", targetAuthorization)
	}
}

func TestDownloadProfilePreservesDefaultRedirectLimit(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Redirect(w, r, "/"+strings.Repeat("x", len(r.URL.Path)), http.StatusFound)
	}))
	defer server.Close()

	c := newClientForURL(t, server.URL)
	_, _, err := c.DownloadProfile(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("DownloadProfile() error = %v", err)
	}
	if got, want := requests.Load(), int32(10); got != want {
		t.Fatalf("redirect requests = %d, want %d", got, want)
	}
}

func TestDownloadProfileCancellationAndShortRead(t *testing.T) {
	t.Run("cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err := newClientForURL(t, server.URL).DownloadProfile(ctx, server.URL)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("DownloadProfile() error = %v, want context.Canceled", err)
		}
	})

	t.Run("short read", func(t *testing.T) {
		c := newClientForURL(t, "https://controller.test")
		c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(&shortReader{data: []byte("partial")}), Request: req}, nil
		})}
		data, finalURL, err := c.DownloadProfile(context.Background(), "https://profile.test/config.yaml")
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("DownloadProfile() error = %v, want io.ErrUnexpectedEOF", err)
		}
		if len(data) != 0 || finalURL != "" {
			t.Fatalf("DownloadProfile() = %q, %q on short read", data, finalURL)
		}
	})
}

func TestDownloadProfileUsesCallerDeadlineNotClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(40 * time.Millisecond)
		_, _ = io.WriteString(w, "profile")
	}))
	defer server.Close()

	client := newClientForURL(t, server.URL)
	client.HTTP.Timeout = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	data, _, err := client.DownloadProfile(ctx, server.URL)
	if err != nil {
		t.Fatalf("DownloadProfile() error = %v", err)
	}
	if got, want := string(data), "profile"; got != want {
		t.Fatalf("DownloadProfile() data = %q, want %q", got, want)
	}
}

func TestDownloadProfileBoundsBodyByActualBytes(t *testing.T) {
	for _, tc := range []struct {
		name          string
		contentLength int64
	}{
		{name: "unknown length", contentLength: -1},
		{name: "misleading length", contentLength: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newClientForURL(t, "https://controller.test")
			c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode:    http.StatusOK,
					ContentLength: tc.contentLength,
					Body:          io.NopCloser(&repeatReader{remaining: MaxBodyBytes + 1}),
					Request:       req,
				}, nil
			})}

			data, finalURL, err := c.DownloadProfile(context.Background(), "https://profile.test/config.yaml")
			if err == nil || !strings.Contains(err.Error(), "exceeds 33554432 bytes") {
				t.Fatalf("DownloadProfile() error = %v", err)
			}
			if len(data) != 0 || finalURL != "" {
				t.Fatalf("DownloadProfile() = %q, %q for oversized body", data, finalURL)
			}
		})
	}
}

func TestDownloadProfileRejectsUnsafeInitialURL(t *testing.T) {
	c := newClientForURL(t, "https://controller.test")
	for _, rawURL := range []string{"/relative", "ftp://profile.test/x", "https://user@profile.test/x"} {
		t.Run(rawURL, func(t *testing.T) {
			if _, _, err := c.DownloadProfile(context.Background(), rawURL); err == nil {
				t.Fatalf("DownloadProfile(%q) succeeded", rawURL)
			}
		})
	}
}

func TestErrorBodiesAreBounded(t *testing.T) {
	c := newClientForURL(t, "https://controller.test")
	c.HTTP = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := bytes.NewBufferString("\x1b]0;owned\x07")
		_, _ = body.WriteString(strings.Repeat("x", int(MaxErrorBytes)+1))
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(body), Request: req}, nil
	})}
	_, err := c.Groups(context.Background())
	if err == nil || strings.Contains(err.Error(), "\x1b") || !strings.Contains(err.Error(), "GET /group") {
		t.Fatalf("Groups() error = %v", err)
	}
	if len(err.Error()) > int(MaxErrorBytes)+len("GET /group: http 502: ")+100 {
		t.Fatalf("error exceeds bounded body allowance: %d bytes", len(err.Error()))
	}
}

func ExampleClient_Warning() {
	c, _ := New("http://controller.test", "secret")
	fmt.Println(c.Warning())
	// Output: plaintext HTTP sends the Mihomo secret
}
