//go:build linux

package api

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const procRouteHeader = "Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT"

func TestParseDefaultRouteInterfaceSelectsLowestMetric(t *testing.T) {
	routes := strings.Join([]string{
		procRouteHeader,
		"not-default 00000001 00000000 0001 0 0 1 00000000 0 0 0",
		"down0 00000000 00000000 0000 0 0 1 00000000 0 0 0",
		"reject0 00000000 00000000 0201 0 0 1 00000000 0 0 0",
		"high0 00000000 00000000 0001 0 0 20 00000000 0 0 0",
		"low0 00000000 00000000 0001 0 0 10 00000000 0 0 0",
	}, "\n")

	iface, err := parseDefaultRouteInterface(strings.NewReader(routes))
	if err != nil {
		t.Fatalf("parseDefaultRouteInterface() error = %v", err)
	}
	if got, want := iface, "low0"; got != want {
		t.Fatalf("parseDefaultRouteInterface() = %q, want %q", got, want)
	}
}

func TestParseDefaultRouteInterfaceRetainsFirstEqualMetric(t *testing.T) {
	routes := procRouteHeader + "\n" +
		"first0 00000000 00000000 0001 0 0 10 00000000 0 0 0\n" +
		"second0 00000000 00000000 0001 0 0 10 00000000 0 0 0"

	iface, err := parseDefaultRouteInterface(strings.NewReader(routes))
	if err != nil {
		t.Fatalf("parseDefaultRouteInterface() error = %v", err)
	}
	if got, want := iface, "first0"; got != want {
		t.Fatalf("parseDefaultRouteInterface() = %q, want %q", got, want)
	}
}

func TestParseDefaultRouteInterfaceRejectsMalformedTables(t *testing.T) {
	validRow := []string{"eth0", "00000000", "00000000", "0001", "0", "0", "10", "00000000", "0", "0", "0"}
	joinRow := func(fields []string) string {
		return strings.Join(fields, " ")
	}
	table := func(row string) string {
		return procRouteHeader + "\n" + row
	}

	tests := []struct {
		name   string
		routes string
	}{
		{name: "empty", routes: ""},
		{name: "bad header", routes: "Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window"},
		{name: "short row", routes: table("eth0 00000000")},
		{name: "extra row column", routes: table(joinRow(append(validRow, "unexpected")))},
		{name: "no default route", routes: table("eth0 00000001 00000000 0001 0 0 10 00000000 0 0 0")},
		{name: "down default route", routes: table("eth0 00000000 00000000 0000 0 0 10 00000000 0 0 0")},
		{name: "rejected default route", routes: table("eth0 00000000 00000000 0201 0 0 10 00000000 0 0 0")},
	}

	for _, numericColumn := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		row := append([]string(nil), validRow...)
		row[numericColumn] = "untrusted-route-content"
		tests = append(tests, struct {
			name   string
			routes string
		}{
			name:   "invalid numeric column " + validRow[numericColumn],
			routes: table(joinRow(row)),
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDefaultRouteInterface(strings.NewReader(tc.routes))
			if err == nil {
				t.Fatal("parseDefaultRouteInterface() succeeded")
			}
			if strings.Contains(err.Error(), "untrusted-route-content") {
				t.Fatalf("parseDefaultRouteInterface() error leaks route content: %q", err)
			}
		})
	}
}

func TestParseDefaultRouteInterfaceRejectsScannerError(t *testing.T) {
	routes := procRouteHeader + "\n" + strings.Repeat("x", 64*1024+1)
	_, err := parseDefaultRouteInterface(strings.NewReader(routes))
	if err == nil {
		t.Fatal("parseDefaultRouteInterface() succeeded")
	}
}

func TestNewInterfaceBoundProfileClient(t *testing.T) {
	client := newInterfaceBoundProfileClient("test0")
	if client == nil {
		t.Fatal("newInterfaceBoundProfileClient() = nil")
	}
	if client.Timeout != 0 {
		t.Fatalf("client Timeout = %v, want 0", client.Timeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("transport Proxy is configured, want nil")
	}
	if !transport.DisableKeepAlives {
		t.Fatal("transport DisableKeepAlives = false, want true")
	}
	if transport.DialContext == nil {
		t.Fatal("transport DialContext = nil")
	}
}

func TestIsDirectProfileUnavailable(t *testing.T) {
	cause := errors.New("operation not permitted")
	bound := &bindToDeviceError{cause: cause}
	wrapped := &url.Error{Op: "Get", URL: "https://profile.test/config.yaml", Err: bound}
	if !isDirectProfileUnavailable(wrapped) {
		t.Fatal("isDirectProfileUnavailable() = false, want true for bind-to-device error")
	}
	if !errors.Is(bound, cause) {
		t.Fatal("bindToDeviceError does not unwrap its cause")
	}
	if got, want := bound.Error(), "direct profile interface binding unavailable"; got != want {
		t.Fatalf("bindToDeviceError.Error() = %q, want %q", got, want)
	}

	ordinary := &url.Error{Op: "Get", URL: "https://profile.test/config.yaml", Err: &net.OpError{Op: "dial", Net: "tcp", Err: cause}}
	if isDirectProfileUnavailable(ordinary) {
		t.Fatal("isDirectProfileUnavailable() = true for ordinary network error")
	}
}
