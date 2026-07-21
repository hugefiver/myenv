//go:build !linux

package api

import "testing"

func TestNewDirectProfileClientUnavailableOffLinux(t *testing.T) {
	client, err := newDirectProfileClient()
	if err != nil {
		t.Fatalf("newDirectProfileClient() error = %v", err)
	}
	if client != nil {
		t.Fatalf("newDirectProfileClient() = %v, want nil", client)
	}
	if isDirectProfileUnavailable(assertionError{}) {
		t.Fatal("isDirectProfileUnavailable() = true, want false")
	}
}

type assertionError struct{}

func (assertionError) Error() string { return "assertion error" }
