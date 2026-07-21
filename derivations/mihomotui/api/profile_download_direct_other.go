//go:build !linux

package api

import "net/http"

func newDirectProfileClient() (*http.Client, error) {
	return nil, nil
}

func isDirectProfileUnavailable(error) bool {
	return false
}
