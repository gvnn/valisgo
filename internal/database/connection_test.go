package database

import (
	"testing"
)

func TestNewConnection_UnsupportedDriver(t *testing.T) {
	_, err := NewConnection("mysql", "dsn")
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}
