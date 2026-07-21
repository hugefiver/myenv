//go:build linux

package api

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	rtfUp     = 0x0001
	rtfReject = 0x0200
)

var (
	errRouteTableMalformed            = errors.New("route table malformed")
	errDefaultRouteUnavailable        = errors.New("default route unavailable")
	errDirectProfileClientUnavailable = errors.New("direct profile client unavailable")
	routeTableColumns                 = []string{"Iface", "Destination", "Gateway", "Flags", "RefCnt", "Use", "Metric", "Mask", "MTU", "Window", "IRTT"}
	routeNumericColumnSpecifications  = []routeNumericColumnSpecification{
		{index: 1, base: 16, bitSize: 32},
		{index: 2, base: 16, bitSize: 32},
		{index: 3, base: 16, bitSize: 32},
		{index: 4, base: 10, bitSize: 64},
		{index: 5, base: 10, bitSize: 64},
		{index: 6, base: 10, bitSize: 64},
		{index: 7, base: 16, bitSize: 32},
		{index: 8, base: 10, bitSize: 64},
		{index: 9, base: 10, bitSize: 64},
		{index: 10, base: 10, bitSize: 64},
	}
)

type routeNumericColumnSpecification struct {
	index   int
	base    int
	bitSize int
}

func parseDefaultRouteInterface(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() || !matchesRouteTableHeader(strings.Fields(scanner.Text())) {
		return "", errRouteTableMalformed
	}

	var (
		defaultInterface string
		lowestMetric     uint64
		found            bool
	)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		values, err := parseRouteTableRow(fields)
		if err != nil {
			return "", err
		}

		destination := values[1]
		flags := values[3]
		metric := values[6]
		mask := values[7]
		if destination != 0 || mask != 0 || flags&rtfUp == 0 || flags&rtfReject != 0 {
			continue
		}
		if !found || metric < lowestMetric {
			defaultInterface = fields[0]
			lowestMetric = metric
			found = true
		}
	}
	if scanner.Err() != nil {
		return "", errRouteTableMalformed
	}
	if !found {
		return "", errDefaultRouteUnavailable
	}
	return defaultInterface, nil
}

func matchesRouteTableHeader(fields []string) bool {
	if len(fields) != len(routeTableColumns) {
		return false
	}
	for index, column := range routeTableColumns {
		if fields[index] != column {
			return false
		}
	}
	return true
}

func parseRouteTableRow(fields []string) ([11]uint64, error) {
	var values [11]uint64
	if len(fields) != len(routeTableColumns) {
		return values, errRouteTableMalformed
	}
	for _, specification := range routeNumericColumnSpecifications {
		value, err := strconv.ParseUint(fields[specification.index], specification.base, specification.bitSize)
		if err != nil {
			return values, errRouteTableMalformed
		}
		values[specification.index] = value
	}
	return values, nil
}

func newDirectProfileClient() (*http.Client, error) {
	routeFile, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, errDirectProfileClientUnavailable
	}
	interfaceName, parseErr := parseDefaultRouteInterface(routeFile)
	closeErr := routeFile.Close()
	if parseErr != nil || closeErr != nil {
		return nil, errDirectProfileClientUnavailable
	}

	interfaceInfo, err := net.InterfaceByName(interfaceName)
	if err != nil || interfaceInfo.Flags&net.FlagUp == 0 {
		return nil, errDirectProfileClientUnavailable
	}
	return newInterfaceBoundProfileClient(interfaceName), nil
}

func newInterfaceBoundProfileClient(interfaceName string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableKeepAlives = true

	dialer := &net.Dialer{
		KeepAlive: -1,
		Control: func(_, _ string, connection syscall.RawConn) error {
			var socketOptionErr error
			if err := connection.Control(func(fd uintptr) {
				socketOptionErr = unix.SetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE, interfaceName)
			}); err != nil {
				return &bindToDeviceError{cause: err}
			}
			if socketOptionErr != nil {
				return &bindToDeviceError{cause: socketOptionErr}
			}
			return nil
		},
	}
	transport.DialContext = dialer.DialContext
	return &http.Client{Transport: transport}
}

type bindToDeviceError struct {
	cause error
}

func (e *bindToDeviceError) Error() string {
	return "direct profile interface binding unavailable"
}

func (e *bindToDeviceError) Unwrap() error {
	return e.cause
}

func isDirectProfileUnavailable(err error) bool {
	var bindErr *bindToDeviceError
	return errors.As(err, &bindErr)
}
